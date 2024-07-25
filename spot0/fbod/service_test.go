package fbod

import (
	"context"
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	awsmock "github.com/doitintl/aws/mocks"
	awspmock "github.com/doitintl/aws/providers/mocks"
	bqmock "github.com/doitintl/bigquery/mocks"
	fsmock "github.com/doitintl/firestore/mocks"
	fsPkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

//go:embed sampleFbodDoc.json
var sampleFbodDoc []byte

//go:embed sampleAsg.json
var sampleAsg []byte

func NewServiceMock(t *testing.T, mockUpdateOdbc bool) (*SpotZeroFbodService, *bqmock.IfcInserter, *fsmock.IFbodFirestore) {
	bqMock := bqmock.NewIfcInserter(t)
	fsMock := fsmock.NewIFbodFirestore(t)
	awsPMock := awspmock.NewIAwsServiceProvider(t)
	awsMock := awsmock.NewIAwsService(t)

	var asg autoscaling.Group
	if err := json.Unmarshal(sampleAsg, &asg); err != nil {
		t.Fatalf("error parsing sample ASG data")
	}

	bqMock.On("Put", mock.Anything, mock.Anything).Return(nil)

	fsMock.On("UpdateFbodStatus", mock.Anything, mock.Anything).Return(nil)
	fsMock.On("GetAccountWithAsgOrRegion", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&fsPkg.Account{
		RoleToAssumeArn: "",
		ExternalID:      "",
	}, nil)

	awsMock.On("GetASG", mock.Anything).Return(&asg, nil)
	awsMock.On("CreateAutoscalingService", mock.Anything, mock.Anything, mock.Anything).Return()

	if mockUpdateOdbc {
		awsMock.On("UpdateAsgMipOdBaseCapacity", mock.Anything, mock.Anything).Return(nil, nil)
	}

	awsPMock.On("GetEmptyService").Return(awsMock)

	return &SpotZeroFbodService{
		loggerProvider:     logger.FromContext,
		BqService:          bqMock,
		FsService:          fsMock,
		AwsServiceProvider: awsPMock,
	}, bqMock, fsMock
}

func TestFbod(t *testing.T) {
	t.Run("SingleFbodCheck - revert odbc", func(t *testing.T) {
		ctx := context.Background()
		svc, bqMock, fsMock := NewServiceMock(t, true)

		var fbodAsg *fsPkg.FbodStatusFsDoc
		if err := json.Unmarshal(sampleFbodDoc, &fbodAsg); err != nil {
			t.Fatalf("error parsing sample FBOD data")
		}

		svc.SingleFbodCheck(ctx, fbodAsg, nil)
		assert.Equal(t, "UpdateFbodStatus", fsMock.Calls[1].Method)
		assert.Equal(t, "Put", bqMock.Calls[0].Method)
		fbodStatus := fsMock.Calls[1].Arguments[1].(*fsPkg.FbodStatusFsDoc)
		assert.Equal(t, fsPkg.FbodStateNormal, fbodStatus.State)
		assert.Equal(t, fsPkg.FbodStateFbod, fbodStatus.PrevState)
		assert.Equal(t, int64(0), *fbodStatus.OdBaseCapacity)
	})

	t.Run("SingleFbodCheck - odbc unchanged", func(t *testing.T) {
		ctx := context.Background()
		svc, bqMock, fsMock := NewServiceMock(t, false)

		var fbodAsg *fsPkg.FbodStatusFsDoc
		if err := json.Unmarshal(sampleFbodDoc, &fbodAsg); err != nil {
			t.Fatalf("error parsing sample FBOD data")
		}

		origOdbc := int64(2)
		fbodAsg.OriginalOdBaseCapacity = &origOdbc

		svc.SingleFbodCheck(ctx, fbodAsg, nil)
		assert.Equal(t, "UpdateFbodStatus", fsMock.Calls[1].Method)
		assert.Equal(t, "Put", bqMock.Calls[0].Method)
		fbodStatus := fsMock.Calls[1].Arguments[1].(*fsPkg.FbodStatusFsDoc)
		assert.Equal(t, fsPkg.FbodStateFbod, fbodStatus.State)
		assert.Equal(t, fsPkg.FbodStateFbod, fbodStatus.PrevState)
		assert.Equal(t, int64(2), *fbodStatus.OdBaseCapacity)
	})
}
