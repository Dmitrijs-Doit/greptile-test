package domain

import "time"

type MessageBQ struct {
	ID           string     `bigquery:"id"`
	Enqueued     time.Time  `bigquery:"enqueued"`
	Sent         *time.Time `bigquery:"sent"`
	Delivered    *time.Time `bigquery:"delivered"`
	Opened       *time.Time `bigquery:"opened"`
	Clicked      *time.Time `bigquery:"clicked"`
	Status       string     `bigquery:"status"`
	Recipient    string     `bigquery:"status"`
	Event        string     `bigquery:"event"`
	Notification string     `bigquery:"notification"`
	Error        *string    `bigquery:"error"`
	Reason       *string    `bigquery:"Reason"`
}
