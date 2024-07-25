package domain

import "time"

type Platforms struct {
	Values []Platform
}

type Platform struct {
	Asset         string `json:"asset"`
	HelperText    string `json:"helperText"`
	Label         string `json:"label"`
	Order         int    `json:"order"`
	Title         string `json:"title"`
	Value         string `json:"value"`
	SaasSupported bool   `json:"saasSupported"`
}

type Product struct {
	Blacklisted bool       `json:"blacklisted"`
	Categories  []Category `json:"categories"`
	ID          string     `json:"id"`
	LastUpdate  time.Time  `json:"lastUpdate" firestore:"last_update"`
	Name        string     `json:"name"`
	Platform    string     `json:"platform"`
	Summary     string     `json:"summary"`
	URL         string     `json:"url"`
	Version     int        `json:"version"`
}

type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
