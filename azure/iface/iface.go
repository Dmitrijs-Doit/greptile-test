package iface

type Payload struct {
	Account        string `json:"account" validate:"required"`
	Container      string `json:"container" validate:"required"`
	ResourceGroup  string `json:"resourceGroup" validate:"required"`
	SubscriptionID string `json:"subscriptionId" validate:"required"`
	Directory      string `json:"directory" validate:"required"`
}

type StorageAccountNameResponse struct {
	StorageAccountName string `json:"storageAccountName"`
}
