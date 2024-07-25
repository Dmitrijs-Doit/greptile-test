package domain

import (
	"errors"
	"strings"
)

type Product string

const (
	ProductFlexsave    Product = "doit-flexsave"
	ProductCostAnomaly Product = "doit-cloud-cost-anomaly-detection"
	ProductDoitConsole Product = "doit-console"

	ProductFlexsaveDevelopment    Product = "doit-flexsave-development"
	ProductCostAnomalyDevelopment Product = "doit-cloud-cost-anomaly-detection-development"
	ProductDoitConsoleDevelopment Product = "doit-console-development"
)

type ProductType string

const (
	ProductTypeFlexsave    ProductType = "flexsave"
	ProductTypeCostAnomaly ProductType = "cost-anomaly"
	ProductTypeDoitConsole ProductType = "doit-console"
)

var (
	ErrProductNotSupported = errors.New("product is not supported")
)

func ExtractProduct(productName string) (Product, error) {
	product := Product(strings.Split(productName, ".")[0])

	switch product {
	case ProductFlexsaveDevelopment:
		return ProductFlexsaveDevelopment, nil
	case ProductFlexsave:
		return ProductFlexsave, nil
	case ProductCostAnomalyDevelopment:
		return ProductCostAnomalyDevelopment, nil
	case ProductCostAnomaly:
		return ProductCostAnomaly, nil
	case ProductDoitConsoleDevelopment:
		return ProductDoitConsoleDevelopment, nil
	case ProductDoitConsole:
		return ProductDoitConsole, nil
	default:
		return "", ErrProductNotSupported
	}
}

func (p Product) ProductType(isProduction bool) (ProductType, error) {
	if isProduction {
		switch p {
		case ProductFlexsave:
			return ProductTypeFlexsave, nil
		case ProductCostAnomaly:
			return ProductTypeCostAnomaly, nil
		case ProductDoitConsole:
			return ProductTypeDoitConsole, nil
		}
	} else {
		switch p {
		case ProductFlexsaveDevelopment:
			return ProductTypeFlexsave, nil
		case ProductCostAnomalyDevelopment:
			return ProductTypeCostAnomaly, nil
		case ProductDoitConsoleDevelopment:
			return ProductTypeDoitConsole, nil
		}
	}

	return "", ErrProductNotSupported
}
