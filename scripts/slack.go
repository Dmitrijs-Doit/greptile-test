package scripts

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	slackgo "github.com/slack-go/slack"

	budgetsDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slack/dal"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/slack"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/slack/iface"
	"github.com/doitintl/slackapi"
	"github.com/doitintl/slackapi/utils"
)

type SlackScripts struct {
	*connection.Connection
	service iface.Slack
	api     *slackapi.SlackAPI
	budgets budgetsDAL.Budgets
}

type slackScriptReq struct {
	CustomerID string `json:"customerId"`
	ChannelID  string `json:"channelId"`
}

func NewSlackScripts(log logger.Provider, conn *connection.Connection) *SlackScripts {
	service, err := slack.NewSlackService(log, conn)
	if err != nil {
		fmt.Println("error initiating SlackScripts, skipping", err)
		return nil
	}

	slackAPI, err := slackapi.NewSlackAPI(context.Background(), common.ProjectID)
	if err != nil {
		fmt.Println("error initiating SlackScripts, skipping", err)
		return nil
	}

	return &SlackScripts{
		conn,
		service,
		slackAPI,
		budgetsDAL.NewBudgetsFirestoreWithClient(conn.Firestore),
	}
}

func (h *SlackScripts) GetSlackWorkspace(ctx *gin.Context) []error {
	errors := []error{}

	var req slackScriptReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	customerID := req.CustomerID

	workspace, workspaceID, userToken, botToken, err := h.service.GetWorkspaceDecrypted(ctx, customerID)
	if err != nil {
		return []error{err}
	}

	fmt.Printf("workspace %s for customer %s:\n%+vֿֿ\nuserToken -> %s\nbotToken -> %s", workspaceID, customerID, workspace, userToken, botToken)

	return errors
}

func (h *SlackScripts) SubscribeExistingSharedChannel(ctx *gin.Context) []error {
	errors := []error{}

	var req slackScriptReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	sharedChannel, err := h.service.SubscribeSharedChannel(ctx, req.CustomerID, req.ChannelID)
	if err != nil {
		return []error{err}
	}

	fmt.Printf("shared channel id: [%s], name: [%s] for customer [%s] is now formally subscribed as a shared channel within our platform", req.ChannelID, sharedChannel.Name, req.CustomerID)

	return errors
}

func (h *SlackScripts) AddBotToChannels(ctx *gin.Context) []error {
	errors := []error{}
	success := 0
	already := 0
	failure := 0

	file, err := os.Open("./scripts/channels.txt")
	if err != nil {
		fmt.Println("no file, running on all shared channels")

		channels, err := h.api.DoitsyClient.GetAllChannels([]string{"public_channel"})
		if err != nil {
			return []error{err}
		}

		sharedChannels := []slackgo.Channel{}

		for _, channel := range channels {
			if channel.IsShared {
				sharedChannels = append(sharedChannels, channel)

				if !channel.IsMember {
					fun := func() error {
						_, err := h.api.YallaClient.InviteUsersToChannel(channel.ID, domain.DoitsyBotUserID)
						return err
					}
					if err := utils.RunWithLimitHandler(fun); err != nil {
						fmt.Println("cannot add", channel.Name)

						failure++

						errors = append(errors, err)

						continue
					}

					success++
				}

				already++
			}
		}

		fmt.Printf("total of %d shared channels out of %d", len(sharedChannels), len(channels))
	} else {
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			channelID := strings.TrimSpace(scanner.Text())
			fun := func() error {
				channel, _, _, err := h.api.DoitsyClient.JoinChannel(channelID)
				if channel != nil {
					fmt.Printf("%s: %s shared %t\n", channel.ID, channel.Name, channel.IsShared)
				}

				return err
			}

			if err := utils.RunWithLimitHandler(fun); err != nil {
				fmt.Println("cannot join", channelID)

				failure++

				errors = append(errors, err)

				continue
			}

			success++
		}
	}

	fmt.Println("success", success, "failures", failure, "already in channel", already)

	return errors
}

func (h *SlackScripts) SetBudgetsSlackChannels(ctx *gin.Context) []error {
	errors := []error{}
	success := 0
	fail := 0

	var req struct {
		CustomerID        string   `json:"customerId"`
		Owners            []string `json:"owners"`
		Label             string   `json:"label"`
		SlackChannelNames []string `json:"slackChannelNames"`
		Override          bool     `json:"override"`
	}
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	budgets, err := h.budgets.ListBudgets(ctx, &budgetsDAL.ListBudgetsArgs{
		CustomerID:     req.CustomerID,
		IsDoitEmployee: true,
		Filter: &budgetsDAL.BudgetListFilter{
			Owners: req.Owners,
		},
	})

	if req.Label != "" {
		filterByLabel := func(allBudgets []budget.Budget, label string) ([]budget.Budget, error) {
			filteredBudgets := []budget.Budget{}
			fs, err := firestore.NewClient(ctx, common.ProjectID)
			if err != nil {
				return nil, err
			}
			labelSnaps, err := fs.Collection("labels").Where("name", "==", req.Label).Documents(ctx).GetAll()
			if err != nil || len(labelSnaps) == 0 {
				return nil, err
			}
			labelId := labelSnaps[0].Ref.ID

			for _, budget := range allBudgets {
				for _, labelRef := range budget.Labels {
					if labelRef.ID == labelId {
						filteredBudgets = append(filteredBudgets, budget)
					}
				}
			}

			return filteredBudgets, nil
		}

		budgets, err = filterByLabel(budgets, req.Label)
		if err != nil {
			return []error{err}
		}
	}

	workspace, _, _, botToken, err := h.service.GetWorkspaceDecrypted(ctx, req.CustomerID)
	if err != nil {
		return []error{err}
	}

	channels := make([]common.SlackChannel, 0)
	for _, channelName := range req.SlackChannelNames {
		channel, err := h.api.Client.WithToken(botToken).GetChannelByName(channelName)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		channels = append(channels, *dal.MapToCommonSlackChannel(channel, workspace))
	}
	fmt.Printf("channels to update: %+v\n", channels)

	for _, budget := range budgets {
		if !req.Override {
			channels = append(channels, budget.RecipientsSlackChannels...)
		}
		budget.RecipientsSlackChannels = channels

		fmt.Printf("updating budget %s\n", budget.Name)
		if err :=
			h.budgets.UpdateBudgetRecipients(ctx, budget.ID, budget.Recipients, budget.RecipientsSlackChannels); err != nil {
			errors = append(errors, err)
			fail++

			continue
		}

		success++
	}

	fmt.Printf("updated %d budgets successfully\n", success)
	fmt.Printf("failed to update %d budgets\n", fail)

	return errors
}
