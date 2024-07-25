package domain

type DeleteEventsReq struct {
	EventIDs []string `json:"eventIds"`
	Clouds   []string `json:"clouds"`
}
