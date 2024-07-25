package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/slack"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/slack/iface"
	"github.com/doitintl/mixpanel"
)

// Slack - handler
type Slack struct {
	loggerProvider logger.Provider
	service        iface.Slack
	mixpanel       *mixpanel.Service
}

func NewSlack(loggerProvider logger.Provider, conn *connection.Connection) *Slack {
	service, err := slack.NewSlackService(loggerProvider, conn)
	if err != nil {
		panic(err)
	}

	return &Slack{
		loggerProvider,
		service,
		mixpanel.NewService(),
	}
}

func (h *Slack) initLogger(ctx *gin.Context, flow string) {
	h.loggerProvider(ctx).SetLabels(map[string]string{
		"service": "slack",
		"flow":    flow,
	})
}

// AcknowledgeAndHandleEvent - sending an ack to slack servers and handling events on a go routine
func (h *Slack) AcknowledgeAndHandleEvent(ctx *gin.Context) error {
	h.initLogger(ctx, "event_subscription")

	body, req, err := h.service.ParseEventSubscriptionRequest(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	responseMessage := req.Challenge
	if req.Event.Type != domain.EventChallenge {
		responseMessage = "event_subscription request received successfully "

		ctxOrigin := ctx.Copy()
		ctxOrigin.Set(domainOrigin.QueryOriginCtxKey, domainOrigin.QueryOriginSlackUnfurl)

		go h.handleEventSubscription(ctxOrigin, body, req)
	}

	// acknowledge slack event or respond with challenge for 1st handshake authentication
	return web.Respond(ctx, responseMessage, http.StatusOK)
}

// handleEventSubscription - handling events triggered by DoiT International Slack app (AF79TTA7N) or Slack Bot (A012TR3MK5E)
func (h *Slack) handleEventSubscription(ctx *gin.Context, body []byte, req *domain.SlackRequest) {
	eventType := req.Event.Type
	l := h.loggerProvider(ctx)
	l.SetLabel("event", string(eventType))

	if err := h.service.ValidateRequest(ctx, body, req.Token); err != nil {
		l.Errorf("error occurred while validating request: %v", err)
		errorreporting.ReportRequestError(ctx, err)

		return
	}

	switch eventType {
	case domain.EventLinkShared:
		mixpanelProperties, err := h.service.HandleLinkSharedEvent(ctx, req)
		if err != nil {
			l.Errorf("error occurred while handling link_shared event: %v", err)
			errorreporting.ReportRequestError(ctx, err)

			return
		}

		if mixpanelProperties != nil {
			if err := h.mixpanel.Import(ctx, mixpanelProperties.Event, mixpanelProperties.Email, mixpanelProperties.Payload); err != nil {
				l.Errorf("mixpanel import error: %v", err)
				errorreporting.ReportRequestError(ctx, err)

				return
			}
		}

	case domain.EventHomeOpened:
		if err := h.service.HandleAppHome(ctx, req); err != nil {
			l.Errorf("error occurred while handling app_home_opened event: %v", err)
			errorreporting.ReportRequestError(ctx, err)

			return
		}

	case domain.EventMemberJoined:
		if err := h.service.HandleUserJoinedSharedChannel(ctx, req.Event); err != nil {
			l.Errorf("error occurred while handling member_joined_channel event: %v", err)
			errorreporting.ReportRequestError(ctx, err)

			return
		}

	default:
		l.Errorf(domain.ErrorUnsupportedEvent, eventType)
	}
}

// OAuth2callback - installation callback for DoiT International Slack app (AF79TTA7N)
func (h *Slack) OAuth2callback(ctx *gin.Context) error {
	h.initLogger(ctx, "installation")

	code := ctx.Query("code")
	state := ctx.Query("state")

	if code == "" {
		return web.NewRequestError(errors.New("Request must contain code field"), http.StatusBadRequest)
	}

	redirectURL, mixpanelProperties, err := h.service.OAuth2callback(ctx, code, state)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	ctx.Redirect(http.StatusFound, redirectURL)

	if mixpanelProperties != nil {
		err = h.mixpanel.Import(ctx, mixpanelProperties.Event, mixpanelProperties.Email, mixpanelProperties.Payload)
		if err != nil {
			return err
		}
	}

	return nil
}

// InstallApp - DoiT International Slack app (AF79TTA7N)
func (h *Slack) InstallApp(ctx *gin.Context) error {
	ctx.Redirect(http.StatusFound, domain.InstallationLink)
	return nil
}

func (h *Slack) SendMixpanelEvent(ctx *gin.Context) error {
	h.initLogger(ctx, "mixpanel")

	mixpanelProperties, err := h.service.ParseMixpanelRequest(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	err = h.mixpanel.Import(ctx, mixpanelProperties.Event, mixpanelProperties.Email, mixpanelProperties.Payload)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// CreateSlackSharedChannel - creating using Doitsy Slack bot (A012TR3MK5E)
func (h *Slack) CreateSlackSharedChannel(ctx *gin.Context) error {
	h.initLogger(ctx, "create-shared-channel")

	res, mixpanelProperties, err := h.service.CreateSlackSharedChannel(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err = h.mixpanel.Import(ctx, mixpanelProperties.Event, mixpanelProperties.Email, mixpanelProperties.Payload); err != nil {
		h.loggerProvider(ctx).Error("mixpanel error: " + err.Error())
	}

	return web.Respond(ctx, res, http.StatusOK)
}

// GetChannelInvitation - get a new invitation link for existing slack channel
func (h *Slack) UpdateCollaboration(ctx *gin.Context) error {
	h.initLogger(ctx, "collaboration")

	var req domain.ChartCollaborationReq
	if err := ctx.ShouldBind(&req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	if err := h.service.UpdateChartCollaboration(ctx, &req); err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

// GetChannelInvitation - get a new invitation link for existing slack channel
func (h *Slack) GetChannelInvitation(ctx *gin.Context) error {
	h.initLogger(ctx, "invitation")

	invitationURL, err := h.service.GetChannelInvitation(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, invitationURL, http.StatusOK)
}

func (h *Slack) GetCustomerChannels(ctx *gin.Context) error {
	h.initLogger(ctx, "get-channels")

	res, err := h.service.GetCustomerChannels(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, res, http.StatusOK)
}

func (h *Slack) AuthTest(ctx *gin.Context) error {
	h.initLogger(ctx, "auth-test")

	res, err := h.service.AuthTest(ctx, ctx.Param("customerID"))
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	return web.Respond(ctx, res, http.StatusOK)
}
