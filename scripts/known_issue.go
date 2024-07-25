package scripts

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/slack"

	"github.com/gin-gonic/gin"
)

type SlackCustomerThread struct {
	ChannelId string `firestore:"channeId"`
	ThreadTs  string `firestore:"threadTs"`
}

type DeleteSlackMessageInput struct {
	IssueId string `json:"issueId"`
}

/*
  - Delete known issue from customers slack shared channels
  - Body:
    {
    "issueId": "p7cvIp8SznvrkZlC6DVT"
    }
*/
func DeleteKnownIssueFromAllSharedChannels(ctx *gin.Context) []error {
	var params DeleteSlackMessageInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	threadsSnap, err := fs.Collection(fmt.Sprintf("knownIssues/%s/customersThreads", params.IssueId)).Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	common.RunConcurrentJobsOnCollection(ctx, threadsSnap, 1, func(ctx context.Context, threadsSnap *firestore.DocumentSnapshot) {
		var d SlackCustomerThread
		if err := threadsSnap.DataTo(&d); err != nil {
			return
		}

		log.Printf("%+v", d)

		slackParams := make(map[string][]string)
		slackParams["channel"] = []string{d.ChannelId}
		slackParams["ts"] = []string{d.ThreadTs}

		respBody, err := slack.Client.Post(ctx.(*gin.Context), "chat.delete", slackParams, nil)
		if err != nil {
			return
		}

		response := struct {
			OK    bool   `json:"ok,omitempty"`
			Error string `json:"error,omitempty"`
		}{}
		if err := json.Unmarshal(respBody, &response); err != nil {
			return
		}

		log.Printf("%+v", response)
	})

	return nil
}
