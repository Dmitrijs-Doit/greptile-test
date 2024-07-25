package dal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func TestNewFirestoreStripeDAL(t *testing.T) {
	ctx := context.Background()
	_, err := NewStripeFirestore(ctx, common.TestProjectID, "")
	assert.NoError(t, err)

	d := NewStripeFirestoreWithClient(nil, "")
	assert.NotNil(t, d)
}
