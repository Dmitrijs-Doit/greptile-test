package service

type DeleteMetricsRequest struct {
	IDs []string `json:"ids" validate:"gt=0,required"`
}
