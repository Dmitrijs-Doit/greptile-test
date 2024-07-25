package domain

type DeleteBatchesReq struct {
	Batches []string `json:"batches"`
}
