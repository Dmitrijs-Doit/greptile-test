package service

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
)

const (
	defaultProduceCode = "DOITINTL"
	defaultProduceDesc = "DoiT Intl invoice"
	other              = "other"
)

// AddLevel3ChargeData adds Level 3 charge data to the Stripe params. Each invoice line item
// is added to the Stripe params as a separate line item using p.AddExtra() method.
func (s *StripeService) AddLevel3ChargeData(
	params *stripe.PaymentIntentParams,
	invoice *invoices.FullInvoice,
	stripeCustomerID string,
	totalAmount int64,
	taxAmount int64,
) error {
	// "merchant_reference" is an alphanumeric string of up to 25 characters in length.
	// This unique value is assigned by the merchant to identify the order.
	// We use the priority ID and invoice ID as the merchant reference.
	merchangeReference := fmt.Sprintf("%s%s", invoice.PriorityID, invoice.ID)

	// "customer_reference" needs to be an alphanumeric string of up to 17 characters in length.
	// Stripe customer ID is normally 18 characters long, with a "cus_" prefix.
	// We remove the prefix and use the rest as the customer reference.
	if !strings.HasPrefix(stripeCustomerID, "cus_") {
		return fmt.Errorf("stripe customer ID %s is not valid", stripeCustomerID)
	}

	productCode, produceDesc := getProductCodeAndDescription(invoice.Products)
	unitCost := totalAmount - taxAmount

	params.AddExtra("level3[merchant_reference]", merchangeReference)
	params.AddExtra("level3[customer_reference]", stripeCustomerID[4:])
	params.AddExtra("level3[line_items][0][product_code]", productCode)
	params.AddExtra("level3[line_items][0][product_description]", produceDesc)
	params.AddExtra("level3[line_items][0][unit_cost]", strconv.FormatInt(unitCost, 10))
	params.AddExtra("level3[line_items][0][quantity]", "1")
	params.AddExtra("level3[line_items][0][tax_amount]", strconv.FormatInt(taxAmount, 10))
	params.AddExtra("level3[line_items][0][discount_amount]", "0")

	return nil
}

// getProductCodeAndDescription returns the product code and description for the given product.
// Product code must be at most 12 characters long.
// Product description must be at most 26 characters long.
func getProductCodeAndDescription(invoiceProducts []string) (string, string) {
	var product string

	if len(invoiceProducts) > 0 {
		for _, p := range invoiceProducts {
			// "other" is a type that can be set on different info line items (like Bucket name)
			if p != other {
				product = p
				break
			}
		}
	}

	switch product {
	// Google products
	case common.Assets.GSuite:
		return "GSUITE", "Google Workspace"
	case common.Assets.GoogleCloud, "looker":
		return "GCP", "Google Cloud"
	case common.Assets.GoogleCloudStandalone:
		return "GCPFSSA", "GCP Flexsave"

	// Amazon products
	case common.Assets.AmazonWebServices:
		return "AWS", "Amazon Web Services"
	case common.Assets.AmazonWebServicesStandalone:
		return "AWSFSSA", "AWS Flexsave"

	// Microsoft products
	case common.Assets.MicrosoftAzure:
		return "MSAZURE", "Microsoft Azure"
	case common.Assets.Office365:
		return "MS365", "Microsoft 365"

	// Other products
	default:
		return defaultProduceCode, defaultProduceDesc
	}
}
