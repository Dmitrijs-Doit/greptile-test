package config

type ExtendedMetric struct {
	Key        string `firestore:"key"`
	Label      string `firestore:"label"`
	Type       string `firestore:"type"`
	Visibility string `firestore:"visibility"`
}

type ExtendedMetrics struct {
	Metrics []ExtendedMetric `firestore:"metrics"`
}
