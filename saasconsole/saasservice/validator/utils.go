package validator

type ValidateAWSSaaSRequest struct {
	AccountID string `json:"accountId"`
	RoleArn   string `json:"roleArn"`
	CURBucket string `json:"curBucket"`
	CURPath   string `json:"curPath"`
}
