package handlers

type EventDTO struct {
	HandleSpecificSink string `json:"handleSpecificSink" validate:"required"`
	DontRun            bool   `json:"dontRun"`
	RemoveData         bool   `json:"removeData"`
}
