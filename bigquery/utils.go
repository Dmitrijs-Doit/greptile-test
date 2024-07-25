package bigquery

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func GetCustomerBQLensCloudConnectDocs(ctx context.Context, fs *firestore.Client, customerID string) ([]*firestore.DocumentSnapshot, error) {
	docSnaps, err := fs.Collection("customers").
		Doc(customerID).
		Collection("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.GoogleCloud).
		Where("categoriesStatus.bigquery-finops", "==", common.CloudConnectStatusTypeHealthy).
		Documents(ctx).
		GetAll()
	if err != nil {
		return nil, err
	}

	return docSnaps, nil
}

func getCustomerBQLensCloudConnectCred(ctx *gin.Context, fs *firestore.Client, customerID string) ([]*common.GoogleCloudCredential, error) {
	docSnaps, err := GetCustomerBQLensCloudConnectDocs(ctx, fs, customerID)
	if err != nil {
		return nil, err
	}

	cloudConnectCreds := make([]*common.GoogleCloudCredential, 0)

	for _, docSnap := range docSnaps {
		var cloudConnectCred common.GoogleCloudCredential

		if err := docSnap.DataTo(&cloudConnectCred); err != nil {
			continue
		}

		cloudConnectCreds = append(cloudConnectCreds, &cloudConnectCred)
	}

	return cloudConnectCreds, nil
}
