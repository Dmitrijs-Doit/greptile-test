package service

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/dal"
)

func generateLogRecord(props *LogRecordProps) map[string]interface{} {
	return map[string]interface{}{
		"email":          props.Email,
		"isDoitEmployee": props.DoitEmployee,
		"success":        false,
		"type":           "subscriptions.order",
		"timestamp":      time.Now(),
		"logLine":        nil,
		"request": map[string]interface{}{
			"body":                 props.RequestBody,
			logger.LabelCustomerID: props.LicenseCustomerID,
			"subscriptionId":       props.SubscriptionID,
		},
		"response": map[string]interface{}{
			"asset":    props.AssetRef,
			"customer": nil,
			"entity":   nil,
			"subscription": map[string]interface{}{
				"before": nil,
				"after":  nil,
			},
		},
	}
}

func startLogListener(ctx context.Context, l logger.ILogger, licenseService dal.ILicense, logChan chan []firestore.Update, done chan bool, props *LogRecordProps) {
	var logRef *firestore.DocumentRef

	var err error

	if props.EnableLog {
		logRecord := generateLogRecord(props)

		logRef, err = licenseService.AddLog(ctx, logRecord)
		if err != nil {
			l.Println(err)
		}
	}

	for {
		select {
		case <-done:
			return
		case updates := <-logChan:
			if !props.EnableLog {
				continue
			}

			_, err := logRef.Update(ctx, updates)

			if err != nil {
				l.Println(err)
			}
		}
	}
}
