package invoices

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log"

	"cloud.google.com/go/storage"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	MaxLatestInvoices = 12
	StatusCanceled    = "Canceled"
	StatusApproved    = "Approved"
)

//go:embed skus.json
var skusB []byte

// ProductSKU :
var ProductSKU map[string]string

// Google Cloud Storage
var (
	StorageClient  *storage.Client
	InvoicesBucket *storage.BucketHandle
)

func init() {
	ProductSKU = make(map[string]string)
	if err := json.Unmarshal(skusB, &ProductSKU); err != nil {
		log.Fatalln(err)
	}

	ctx := context.Background()

	gcs, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	StorageClient = gcs
	InvoicesBucket = StorageClient.Bucket(fmt.Sprintf("%s-priority", common.ProjectID))
}
