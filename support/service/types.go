package service

type PlatformAPI struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type PlatformsAPI struct {
	Platforms []PlatformAPI `json:"platforms"`
}

type ProductAPI struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
	Platform    string `json:"platform"`
}

type ProductsAPI struct {
	Products []ProductAPI `json:"products"`
}

type SwaggProductsRequestData struct {
	Platform string `json:"platform"`
}
