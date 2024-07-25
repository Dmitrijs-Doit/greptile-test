package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SentNotificationsTracker interface {
	GetSent(ctx context.Context) (map[string]string, error)
	UpdateSent(ctx context.Context, sent map[string]string) error
	GetLastProcessedDate(ctx context.Context) (time.Time, error)
	UpdateLastProcessedDate(ctx context.Context, date time.Time) error
}

type FsTracker struct {
	DocRef *firestore.DocumentRef
}

type SentNotificationsDoc struct {
	Sent          map[string]string `firestore:"sent"`
	LastProcessed time.Time         `firestore:"lastProcessed"`
}

func (d *FsTracker) GetSent(ctx context.Context) (map[string]string, error) {
	doc, err := d.DocRef.Get(ctx)
	if err != nil {
		// if first time running, there is no document
		if status.Code(err) == codes.NotFound {
			return make(map[string]string), nil
		}

		return nil, err
	}

	docData := &SentNotificationsDoc{}
	if err := doc.DataTo(&docData); err != nil {
		return nil, err
	}

	return docData.Sent, nil
}

func (d *FsTracker) UpdateSent(ctx context.Context, sent map[string]string) error {
	_, err := d.DocRef.Set(ctx, &SentNotificationsDoc{
		Sent: sent,
	}, firestore.Merge([]string{"sent"}))

	return err
}

func (d *FsTracker) GetLastProcessedDate(ctx context.Context) (time.Time, error) {
	doc, err := d.DocRef.Get(ctx)
	if err != nil {
		// if first time running, there is no document
		if status.Code(err) == codes.NotFound {
			return time.Now(), nil
		}

		return time.Time{}, err
	}

	docData := &SentNotificationsDoc{}
	if err := doc.DataTo(&docData); err != nil {
		return time.Time{}, err
	}

	return docData.LastProcessed, nil
}

func (d *FsTracker) UpdateLastProcessedDate(ctx context.Context, date time.Time) error {
	_, err := d.DocRef.Set(ctx, &SentNotificationsDoc{
		LastProcessed: date,
	}, firestore.Merge([]string{"lastProcessed"}))

	return err
}
