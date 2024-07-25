package doitproducts

import (
	"fmt"
	"strings"
)

//S-ST-M-D-002 - Standard Tier Variable Fee - GCP
//S-ST-M-D-003 - Standard Tier Variable Fee - AWS
//S-ST-M-D-004 - Standard Tier Variable Fee - Azure
//S-ST-M-D-005 - Standard Tier Variable Fee - Looker
//S-ST-M-D-006 - Standard Tier Variable Fee - Office 365
//S-ST-M-D-007 - Standard Tier Variable Fee - Google Workspace

var cloudSku = map[string]string{
	"google-cloud":          "-002",
	"google-cloud-platform": "-002",
	"amazon-web-services":   "-003",
	"azure":                 "-004",
	"looker":                "-005",
	"office-365":            "-006",
	"google-workplace":      "-007",
}

var doitProductsSku = map[string]string{
	"navigator-standard-fixed-fee":   "P-ST-M-D-001",
	"navigator-enhanced-fixed-fee":   "P-ET-M-D-001",
	"navigator-premium-fixed-fee":    "P-PT-M-D-001",
	"navigator-enterprise-fixed-fee": "P-EP-M-D-001",

	"solve-standard-fixed-fee":   "S-ST-M-D-001", //"solve-standard-variable-fee": "S-ST-M-D-002" will be generated
	"solve-enhanced-fixed-fee":   "S-ET-M-D-001", // "solve-enhanced-variable-fee": "S-ET-M-D-002", will be generated
	"solve-premium-fixed-fee":    "S-PT-M-D-001", // "solve-premium-variable-fee": "S-PT-M-D-002", will be generated
	"solve-enterprise-fixed-fee": "S-EP-M-D-001", // "solve-enterprise-variable-fee": "S-PT-M-D-002", will be generated
}

var singleSaleSku = map[string]string{
	"aws-data-to-insights":   "-001",
	"AWS - Data to Insights": "-001",
	"aws-gen-ai":             "-002",
	"AWS - GenAI":            "-002",
	"aws-kubernetes":         "-003",
	"AWS - Kubernetes":       "-003",
	"gcp-data-to-insights":   "-101",
	"GCP - Data to Insights": "-101",
	"gcp-get-ai":             "-102",
	"GCP - GenAI":            "-102",
	"gcp-kubernetes":         "-103",
	"GCP - Kubernetes":       "-103",
}

func getFixedSkuID(packageType, packageName, paymentTerm, pointOfSale string) (string, error) {
	baseSKU, ok := doitProductsSku[packageType+"-"+packageName+"-fixed-fee"]
	if !ok {
		return "", fmt.Errorf("SKU ID Not found for packageType %v - packageName %v - paymentTerm %v - pointOfSale %v", packageType, packageName, paymentTerm, pointOfSale)
	}

	if paymentTerm == "annual" {
		baseSKU = baseSKU[:5] + "A" + baseSKU[6:]
	}

	if pointOfSale == "aws-marketplace" || pointOfSale == "gcp-marketplace" {
		baseSKU = baseSKU[:7] + "R" + baseSKU[8:]
	}

	return baseSKU, nil
}

func getVariableSkuID(packageType, packageName, paymentTerm, pointOfSale string, cloud string) (string, error) {
	baseSKU, err := getFixedSkuID(packageType, packageName, paymentTerm, pointOfSale)
	if err != nil {
		return "", err
	}

	cloudSuffix, prs := cloudSku[cloud]
	if !prs {
		return "", fmt.Errorf("SKU ID Not found for packageType %v - packageName %v - paymentTerm %v - pointOfSale %v - cloud %v",
			packageType, packageName, paymentTerm, pointOfSale, cloud)
	}

	return strings.Replace(baseSKU, "-001", cloudSuffix, 1), nil

}

func getSingleSaleSkuID(pointOfSale, skuNumber string) (string, error) {
	sku := "S-SAC-M-"

	if pointOfSale == "aws-marketplace" || pointOfSale == "gcp-marketplace" {
		sku += "R"
	} else {
		sku += "D"
	}

	//suffix, ok := singleSaleSku[acceleratorLabel]
	//if !ok {
	//	return "", fmt.Errorf("SKU ID Not found for packageType %v - packageName %v", packageType, acceleratorLabel)
	//}

	sku += "-" + skuNumber

	return sku, nil
}
