package domain

import (
	"time"

	"cloud.google.com/go/firestore"
)

type UserCustomer struct {
	Ref       *firestore.DocumentRef `firestore:"ref"`
	Name      string                 `firestore:"name"`
	LowerName string                 `firestore:"_name"`
}

// User represents a firestore user document
type User struct {
	Customer           UserCustomer             `firestore:"customer"`
	Domain             string                   `firestore:"domain"`
	Entities           []*firestore.DocumentRef `firestore:"entities"`
	Enrichment         interface{}              `firestore:"enrichment"`
	Permissions        []string                 `firestore:"permissions"`
	Notifications      []int64                  `firestore:"userNotifications"`
	Role               interface{}              `firestore:"role"`
	JobFunction        interface{}              `firestore:"jobFunction"`
	Email              string                   `firestore:"email"`
	DisplayName        string                   `firestore:"displayName"`
	FirstName          string                   `firestore:"firstName"`
	LastName           string                   `firestore:"lastName"`
	Roles              []*firestore.DocumentRef `firestore:"roles"`
	AccessKey          string                   `firestore:"accessKey"`
	Organizations      []*firestore.DocumentRef `firestore:"organizations"`
	EmailNotifications []string                 `firestore:"emailNotifications"`
	DailyDigests       []*firestore.DocumentRef `firestore:"dailyDigests"`
	WeeklyDigests      []*firestore.DocumentRef `firestore:"weeklyDigests"`
	TermsOfService     *TermsOfService          `firestore:"termsOfService"`
	ID                 string                   `firestore:"-"`
}

type TermsOfService struct {
	TimeCreated time.Time `firestore:"timeCreated"`
	Type        string    `firestore:"type"`
}
