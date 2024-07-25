package domain

type CreateCustomerReq struct {
	Cloud []string `json:"cloud"  binding:"required"`
}
