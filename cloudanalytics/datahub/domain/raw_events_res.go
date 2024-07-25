package domain

type AddRawEventsRes struct {
	EventsCount int  `json:"eventsCount"`
	Execute     bool `json:"execute"`
}
