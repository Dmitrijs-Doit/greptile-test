package common

import (
	"time"

	"cloud.google.com/go/firestore"
)

// ReportStatus represents a firestore reportStatus document
type ReportStatusData struct {
	Customer          *firestore.DocumentRef `firestore:"customer"`
	Status            map[string]StatusInfo  `firestore:"status"`
	TimeModified      time.Time              `firestore:"timeModified,serverTimestamp"`
	OverallLastUpdate time.Time              `firestore:"overallLastUpdate"`
}

type StatusInfo struct {
	LastUpdate time.Time `firestore:"lastUpdate,omitempty"`
	Status     string    `firestore:"status,omitempty"`
}

type ReportStatus struct {
	Status map[string]StatusInfo
}

type ReportStatusType string

const (
	GoogleCloudReportStatus ReportStatusType = "google-cloud"
	AWSReportStatus         ReportStatusType = "amazon-web-services"
	GKEReportStatus         ReportStatusType = "gke"
)

type GKEStatus string

const (
	GKEStatusEnabled      GKEStatus = "enabled"
	GKEStatusIncomplete   GKEStatus = "incomplete"
	GKEStatusUnconfigured GKEStatus = "unconfigured"
)
