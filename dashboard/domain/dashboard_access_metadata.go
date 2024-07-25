package domain

import "time"

type DashboardAccessMetadata struct {
	CustomerID        string     `firestore:"customerId"`
	OrganizationID    string     `firestore:"organizationId"`
	DashboardID       string     `firestore:"dashboardId"`
	TimeLastAccessed  *time.Time `firestore:"timeLastAccessed"`
	TimeLastRefreshed *time.Time `firestore:"timeLastRefreshed"`
}
