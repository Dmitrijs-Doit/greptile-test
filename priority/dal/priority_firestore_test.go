package dal

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
	testPackage "github.com/doitintl/tests"
)

func TestPriorityFirestore_HandleAvalaraStatus(t *testing.T) {
	if err := testPackage.LoadTestData("Priority"); err != nil {
		t.Error(err)
	}

	ctx := context.Background()

	d, err := NewPriorityFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	shouldPingAvalara, healthy, err := d.HandleAvalaraStatus(ctx)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, true, shouldPingAvalara)
	assert.Equal(t, true, healthy)
	// case2: ping occurred less than 10 minutes ago
	_ = d.SetAvalaraHealthyStatus(ctx, true)

	shouldPingAvalara2, healthy2, err := d.HandleAvalaraStatus(ctx)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, false, shouldPingAvalara2)
	assert.Equal(t, true, healthy2)
}

func TestPriorityFirestore_SetAvalaraHealthyStatus(t *testing.T) {
	if err := testPackage.LoadTestData("Priority"); err != nil {
		t.Error(err)
	}

	ctx := context.Background()

	d, err := NewPriorityFirestore(ctx, common.TestProjectID)
	if err != nil {
		t.Error(err)
	}

	err = d.SetAvalaraHealthyStatus(ctx, true)
	if err != nil {
		t.Error(err)
	}

	docSnap, err := d.getAvalaraStatusRef(ctx).Get(ctx)
	if err != nil {
		t.Error(err)
	}

	var avalaraStatus priorityDomain.AvalaraStatus
	if err := docSnap.DataTo(&avalaraStatus); err != nil {
		t.Error(err)
	}

	assert.Equal(t, true, avalaraStatus.Healthy)
	assert.Equal(t, false, avalaraStatus.Locked)
	assert.Equal(t, true, time.Since(avalaraStatus.Timestamp) < avalaraMinPingInterval)
}
