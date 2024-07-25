package firebase

import (
	"context"

	"cloud.google.com/go/firestore"
)

type AutomaticWriteBatch struct {
	fs      *firestore.Client
	batches []*firestore.WriteBatch
	i       int
	size    int
	limit   int
}

// Note - please use BatchProvider on future usage when required (refer to server/shared/firestore/batch_provider.go)

// NewAutomaticWriteBatch allows making firestore batch requests with more than 500
// write operation without keeping track of the batch size
func NewAutomaticWriteBatch(fs *firestore.Client, limit int) *AutomaticWriteBatch {
	if limit <= 0 || limit > 500 {
		limit = 500
	}

	return &AutomaticWriteBatch{
		fs:      fs,
		batches: []*firestore.WriteBatch{fs.Batch()},
		i:       0,
		size:    0,
		limit:   limit,
	}
}

func (b *AutomaticWriteBatch) Commit(ctx context.Context) []error {
	var errors []error

	for i, batch := range b.batches {
		if i == b.i && b.size == 0 {
			break
		}

		_, err := batch.Commit(ctx)
		if err != nil {
			errors = append(errors, err)
		}
	}

	// Reset batch
	b.i = 0
	b.size = 0
	b.batches = []*firestore.WriteBatch{b.fs.Batch()}

	return errors
}

func (b *AutomaticWriteBatch) addOperation() *AutomaticWriteBatch {
	b.size++
	if b.size >= b.limit {
		b.batches = append(b.batches, b.fs.Batch())
		b.size = 0
		b.i++
	}

	return b
}

// Create adds a Create operation to the batch.
// See DocumentRef.Create for details.
func (b *AutomaticWriteBatch) Create(dr *firestore.DocumentRef, data interface{}) *AutomaticWriteBatch {
	b.batches[b.i].Create(dr, data)
	return b.addOperation()
}

// Set adds a Set operation to the batch.
// See DocumentRef.Set for details.
func (b *AutomaticWriteBatch) Set(dr *firestore.DocumentRef, data interface{}, opts ...firestore.SetOption) *AutomaticWriteBatch {
	b.batches[b.i].Set(dr, data, opts...)
	return b.addOperation()
}

// Delete adds a Delete operation to the batch.
// See DocumentRef.Delete for details.
func (b *AutomaticWriteBatch) Delete(dr *firestore.DocumentRef, opts ...firestore.Precondition) *AutomaticWriteBatch {
	b.batches[b.i].Delete(dr, opts...)
	return b.addOperation()
}

// Update adds an Update operation to the batch.
// See DocumentRef.Update for details.
func (b *AutomaticWriteBatch) Update(dr *firestore.DocumentRef, data []firestore.Update, opts ...firestore.Precondition) *AutomaticWriteBatch {
	b.batches[b.i].Update(dr, data, opts...)
	return b.addOperation()
}
