package rowsvalidator

import (
	"time"
)

type RowsValidatorMetadata struct {
	LastValidated *time.Time `firestore:"lastValidated"`
}
