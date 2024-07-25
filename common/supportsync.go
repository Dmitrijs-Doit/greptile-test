package common

import "time"

// Service object represent each service supported by CRE's
type Service struct {
	ID          string     `json:"id" firestore:"id"`
	Name        string     `json:"name" firestore:"name"`
	Summary     string     `json:"summary" firestore:"summary"`
	URL         string     `json:"url" firestore:"url"`
	Categories  []Category `json:"categories" firestore:"categories"`
	Tags        []string   `json:"tags" firestore:"tags"`
	Platform    string     `firestore:"platform"`
	LastUpdate  time.Time  `firestore:"last_update"`
	Version     int64      `firestore:"version"`
	BlackListed bool       `firestore:"blacklisted"`
}

type Category struct {
	ID   string `json:"id" firestore:"id"`
	Name string `json:"name" firestore:"name"`
}
