package domain

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/doitintl/validator"
)

type LookerContractProperties struct {
	ContractDuration int64               `json:"contractDuration" validate:"required" firestore:"contractDuration"`
	InvoiceFrequency int64               `json:"invoiceFrequency" validate:"required" firestore:"invoiceFrequency"`
	SalesProcess     string              `json:"salesProcess" validate:"required" firestore:"salesProcess"`
	Skus             []LookerContractSKU `json:"skus" validate:"required" firestore:"skus"`
}
type LookerContractSKU struct {
	MonthlySalesPrice float64               `json:"monthlySalesPrice" validate:"required" firestore:"monthlySalesPrice"`
	Months            int64                 `json:"months" validate:"required" firestore:"months"`
	Quantity          int64                 `json:"quantity" validate:"required" firestore:"quantity"`
	SkuName           LookerContractSKUName `json:"skuName" validate:"required" firestore:"skuName"`
	StartDate         time.Time             `json:"startDate" validate:"required" firestore:"startDate"`
}

type LookerContractSKUName struct {
	GoogleSKU        string  `json:"googleSku" validate:"required" firestore:"googleSku"`
	Label            string  `json:"label" validate:"required" firestore:"label"`
	MonthlyListPrice float64 `json:"monthlyListPrice" validate:"required" firestore:"monthlyListPrice"`
}

func (l *LookerContractProperties) DecodePropertiesMapIntoStruct(props map[string]interface{}) (LookerContractProperties, error) {
	var lookerProps LookerContractProperties

	propsJSON, err := json.Marshal(props)
	if err != nil {
		return lookerProps, nil
	}

	if err := validator.UnmarshalJSON(propsJSON, &lookerProps); err != nil {
		return lookerProps, fmt.Errorf("error unmarshalling properties for looker contract %s", err.Error())
	}

	return lookerProps, nil
}

type UpdateTableInterval struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}
