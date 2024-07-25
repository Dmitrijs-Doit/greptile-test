package domain

type CatalogItem struct {
	SkuID   string           `firestore:"skuId"`
	Plan    string           `firestore:"plan"`
	Payment string           `firestore:"payment"`
	Price   CatalogItemPrice `firestore:"price"`
}

type CatalogItemPrice struct {
	USD float64 `firestore:"USD"`
	EUR float64 `firestore:"EUR"`
	GBP float64 `firestore:"GBP"`
	AUD float64 `firestore:"AUD"`
}
