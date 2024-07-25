package algolia

type Config struct {
	AppID             string   `json:"appId"`
	SearchKey         string   `json:"searchKey"`
	RestrictedIndices []string `json:"restrictedIndices"`
}

type FirestoreConfig struct {
	Config
	DevAppID          string   `json:"devAppId"`
	DevSearchKey      string   `json:"devSearchKey"`
}
