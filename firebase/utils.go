package firebase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"google.golang.org/api/iterator"
)

type documentUpdateFunc func(*firestore.DocumentSnapshot, interface{}) (map[string]interface{}, error)

// GetUniqDocSnaps - return slice of uniq docsnaps by Ref.ID
func GetUniqDocSnaps(docSnaps []*firestore.DocumentSnapshot) []*firestore.DocumentSnapshot {
	ids := make(map[string]bool)
	list := make([]*firestore.DocumentSnapshot, 0)

	for _, doc := range docSnaps {
		if _, id := ids[doc.Ref.ID]; !id {
			ids[doc.Ref.ID] = true

			list = append(list, doc)
		}
	}

	return list
}

// GetFirestoreRelativePath - return docRef path without ID and without project prefex
func GetFirestoreRelativePath(fullPath string) string {
	collectionPathWithDoc := strings.Split(fullPath, "documents/")
	collectionPath := strings.Split(collectionPathWithDoc[1], "/")
	collectionPath = collectionPath[:len(collectionPath)-1]
	collectionPathString := strings.Join(collectionPath, "/")

	return collectionPathString
}

// CopyAllSubCollectionsDocuments copies all documents in all subcollections of docSnapFrom to origDocSnapTo
// destinationBranch is array of consequent collection/document names to be created
// upon origDocSnapTo to accomodate the copied data
func CopyAllSubCollectionsDocuments(ctx context.Context, wb iface.Batch, docRefFrom *firestore.DocumentRef, origDocSnapTo *firestore.DocumentRef, destinationBranch []string, updateFn documentUpdateFunc, auxData interface{}) error {
	if len(destinationBranch)%2 != 0 {
		return errors.New("invalid destination branch")
	}

	collections, err := docRefFrom.Collections(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, collection := range collections {
		documents, err := collection.Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		for _, document := range documents {
			docSnapTo := origDocSnapTo
			for i := 0; i < len(destinationBranch); i += 2 {
				docSnapTo = docSnapTo.Collection(destinationBranch[i]).Doc(destinationBranch[i+1])
			}

			newDestBranch := []string{}

			var data map[string]interface{}

			if updateFn != nil {
				if data, err = updateFn(document, auxData); err != nil {
					return err
				}
			} else {
				data = document.Data()
			}

			if len(data) != 0 {
				if wb != nil {
					if err = wb.Set(ctx, docSnapTo.Collection(collection.ID).Doc(document.Ref.ID), data); err != nil {
						return err
					}
				} else {
					_, err = docSnapTo.Collection(collection.ID).Doc(document.Ref.ID).Set(ctx, data)
					if err != nil {
						return err
					}
				}
			} else {
				newDestBranch = []string{collection.ID, document.Ref.ID}
			}

			if err := CopyAllSubCollectionsDocuments(ctx, wb, document.Ref, docSnapTo, newDestBranch, updateFn, auxData); err != nil {
				return err
			}
		}
	}

	return nil
}

func IsNoFieldError(err error, field string) bool {
	return err.Error() == fmt.Sprintf("firestore: no field %q", field)
}

func ExecuteQueries(ctx context.Context, queries []firestore.Query) ([]*firestore.DocumentSnapshot, error) {
	var (
		docSnaps []*firestore.DocumentSnapshot
		wg       sync.WaitGroup
		mu       sync.Mutex
		errs     []error
	)

	l := logger.FromContext(ctx)

	errCh := make(chan error, len(queries))

	for _, q := range queries {
		wg.Add(1)

		go func(q firestore.Query) {
			defer wg.Done()

			iter := q.Documents(ctx)

			for {
				doc, err := iter.Next()
				if err == iterator.Done {
					break
				}

				if err != nil {
					errCh <- err
					return
				}

				mu.Lock()
				docSnaps = append(docSnaps, doc)
				mu.Unlock()
			}
		}(q)
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		l.Error(errs)

		return nil, errors.New("failed to execute query")
	}

	return docSnaps, nil
}
