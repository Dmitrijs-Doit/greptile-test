package invoicing

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type CatalogItem struct {
	SkuID            string            `firestore:"skuId"`
	SkuName          string            `firestore:"skuName"`
	SkuPriority      string            `firestore:"skuPriority"`
	Plan             string            `firestore:"plan"`
	Payment          string            `firestore:"payment"`
	Price            CatalogItemPrice  `firestore:"price"`
	PrevPrice        *CatalogItemPrice `firestore:"prevPrice"`
	PrevPriceEndDate *time.Time        `firestore:"prevPriceEndDate"`
}

type CatalogItemPrice struct {
	USD *float64 `firestore:"USD"`
	EUR *float64 `firestore:"EUR"`
	GBP *float64 `firestore:"GBP"`
	AUD *float64 `firestore:"AUD"`
	BRL *float64 `firestore:"BRL"`
	NOK *float64 `firestore:"NOK"`
	DKK *float64 `firestore:"DKK"`
}

const (
	CreditRank            int = 999998
	InvoiceAdjustmentRank int = 999999

	InvoicingInfoSKU            = "INFOSKU"
	AmazonWebServicesSKU        = "AMWESEF"
	ChicagoAmazonWebServicesSKU = "AMWESEFCH"
	GoogleCloudSKU              = "GOCLPLF"
	ChicagoGoogleCloudSKU       = "GOCLPLFCH"
	MicrosoftAzureSKU           = "MIAZURE"
	GSuiteSKU                   = "ADJGSUT"
	Office365SKU                = "ADJO365"
	FlexsaveAwsStandadaloneSKU  = "AWSFSSA"
	FlexsaveGcpStandadaloneSKU  = "GCPFSSA"
)

var gsuiteCatalog []*CatalogItem
var office365Catalog []*CatalogItem

func init() {
	ctx := context.Background()

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		log.Fatalln(err)
	}

	defer fs.Close()

	if err := initGSuiteCatalog(ctx, fs); err != nil {
		log.Fatalln(err)
	}

	if err := initOffice365Catalog(ctx, fs); err != nil {
		log.Fatalln(err)
	}
}

func initGSuiteCatalog(ctx context.Context, fs *firestore.Client) error {
	gsuiteCatalog = make([]*CatalogItem, 0)

	iter := fs.Collection("catalog").Doc("g-suite").Collection("services").Documents(ctx)

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			return nil
		}

		if err != nil {
			return err
		}

		var item CatalogItem
		if err := docSnap.DataTo(&item); err != nil {
			return err
		}

		gsuiteCatalog = append(gsuiteCatalog, &item)
	}
}

func initOffice365Catalog(ctx context.Context, fs *firestore.Client) error {
	office365Catalog = make([]*CatalogItem, 0)

	iter := fs.Collection("catalog").Doc("office-365").Collection("services").Documents(ctx)

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			return nil
		}

		if err != nil {
			return err
		}

		var item CatalogItem
		if err := docSnap.DataTo(&item); err != nil {
			return err
		}

		office365Catalog = append(office365Catalog, &item)
	}
}
