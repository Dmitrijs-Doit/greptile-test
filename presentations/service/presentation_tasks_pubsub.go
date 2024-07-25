package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/pubsub"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
	"github.com/gin-gonic/gin"
)

const doneMetaDataAssetsTopic = "presentation-update-metadata-assets-done"
const doneMetaDataAssetsSubscription = "presentation-update-metadata-assets-done-subscription"

type JobSyncMessage = struct {
	AssetType    string `json:"assetType"`
	Success      bool   `json:"success"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	SessionID    string `json:"sessionId,omitempty"`
}

type SafeMessageSlice struct {
	sync.RWMutex
	messages           map[string]JobSyncMessage
	requiredAssetTypes []string
}

func (sms *SafeMessageSlice) WithRequiredAssetTypes(assetTypes []string) {
	sms.messages = make(map[string]JobSyncMessage)
	sms.requiredAssetTypes = assetTypes
}

func (sms *SafeMessageSlice) AppendMessage(msg JobSyncMessage) {
	sms.Lock()
	sms.messages[msg.AssetType] = msg
	sms.Unlock()
}

func (sms *SafeMessageSlice) isDone() bool {
	for _, assetType := range sms.requiredAssetTypes {
		if _, ok := sms.messages[assetType]; !ok {
			return false
		}
	}

	return true
}

func (sms *SafeMessageSlice) hasFailed() bool {
	for _, value := range sms.messages {
		if !value.Success {
			return true
		}
	}

	return false
}

func (sms *SafeMessageSlice) getError() error {
	errorMessages := make([]string, 0)

	for _, value := range sms.messages {
		errorMessages = append(errorMessages, value.ErrorMessage)
	}

	return fmt.Errorf("Synced errors: %s", strings.Join(errorMessages, ", "))
}

func (p *PresentationService) DispatchCloudTask(ctx *gin.Context, path string, queue common.TaskQueue) error {
	l := p.Logger(ctx)

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   path,
		Queue:  queue,
	}

	l.Printf("Creating task for path: %s", path)

	if _, err := p.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(nil)); err != nil {
		return fmt.Errorf("failed to create presentation task: %s %w", path, err)
	}

	l.Printf("Task created for path: %s", path)

	return nil
}

func (p *PresentationService) CreateWidgetUpdateTasks(ctx *gin.Context) error {
	l := p.Logger(ctx)
	l.SetLabel(log.LabelPresentationUpdateStage.String(), "widgets")

	customers, err := p.customersDAL.GetPresentationCustomers(ctx)
	if err != nil {
		return fmt.Errorf(FetchCustomerErr, err)
	}

	for _, customer := range customers {
		l.Infof("Azure billing update data for customer: %s", customer.ID)

		if err = p.DispatchCloudTask(ctx, fmt.Sprintf("/tasks/analytics/widgets/customers/%s?orgId=root", customer.ID), common.TaskQueuePresentationAWS); err != nil {
			return err
		}
	}

	return nil
}

func (p *PresentationService) SendErrorSyncMessage(ctx *gin.Context, asset string, err error) error {
	return p.sendSyncMessage(ctx, &JobSyncMessage{
		AssetType:    asset,
		Success:      false,
		ErrorMessage: err.Error(),
		SessionID:    ctx.Query(sessionIDQueryParamName),
	})
}

func (p *PresentationService) SendDoneSyncMessage(ctx *gin.Context, asset string) error {
	return p.sendSyncMessage(ctx, &JobSyncMessage{
		AssetType:    asset,
		Success:      true,
		ErrorMessage: "",
		SessionID:    ctx.Query(sessionIDQueryParamName),
	})
}

func (p *PresentationService) sendSyncMessage(ctx *gin.Context, message *JobSyncMessage) error {
	client := p.conn.Pubsub(ctx)

	msg, err := json.Marshal(message)
	if err != nil {
		return err
	}

	res := client.Topic(doneMetaDataAssetsTopic).Publish(ctx, &pubsub.Message{
		Data: msg,
	})

	_, err = res.Get(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (p *PresentationService) ReceiveDoneUpdateMetadataAssetsMessage(ctx *gin.Context, assetTypes []string, sessionID string) error {
	l := p.Logger(ctx)
	client := p.conn.Pubsub(ctx)
	sub := client.Subscription(doneMetaDataAssetsSubscription)

	ctxWithTimeout, cancel := context.WithTimeout(ctx.Request.Context(), time.Hour*2)
	defer cancel()

	var wg sync.WaitGroup

	sms := &SafeMessageSlice{}
	sms.WithRequiredAssetTypes(assetTypes)

	errs := make(chan error)

	l.Printf("Receiving pubsub done message")

	wg.Add(1)

	go func() {
		err := sub.Receive(ctxWithTimeout, func(ctx context.Context, pubsubMsg *pubsub.Message) {
			var message JobSyncMessage

			err := json.Unmarshal(pubsubMsg.Data, &message)
			if err != nil {
				l.Printf("Error parsing pubsub message: %s \n %s", err, string(pubsubMsg.Data))

				// just a temporary silent fail to skip old messages
				if string(pubsubMsg.Data) == "gcp-done" {
					return
				}

				errs <- err
			}

			if message.SessionID != sessionID {
				return
			}

			l.Printf("[current session ID: %s] Received pubsub done message %+v\n", sessionID, message)

			pubsubMsg.Ack()

			sms.AppendMessage(message)

			if sms.isDone() {
				wg.Done()

				if sms.hasFailed() {
					errs <- sms.getError()
				}
			}
		})
		if err != nil {
			wg.Done()
			l.Printf("Error receiving pubsub done message: %v", err)
			errs <- err
		}
	}()

	wg.Wait()

	defer close(errs)

	l.Printf("Received pubsub done message")

	select {
	case err := <-errs:
		return err
	default:
		return nil
	}
}
