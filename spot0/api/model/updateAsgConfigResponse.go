package model

type UpdateAsgConfigResponse struct {
	Success   bool   `json:"success"`
	ErrorDesc string `json:"errorDesc"`
	ErrorCode string `json:"errorCode"`
}
