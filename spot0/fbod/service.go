package fbod

import (
	"context"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"

	doitAws "github.com/doitintl/aws"
	doitAwsIface "github.com/doitintl/aws/iface"
	sharedAwsPkg "github.com/doitintl/aws/pkg"
	"github.com/doitintl/aws/providers"
	doitBq "github.com/doitintl/bigquery"
	doitBqIface "github.com/doitintl/bigquery/iface"
	doitFs "github.com/doitintl/firestore"
	fsPkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	doitSm "github.com/doitintl/secretmanager"
)

const (
	AwsSpot0RoleSecretName = "aws-spot0-role"
	Spot0AwsCredSecretName = "spot0-aws-cred"
	EnvVarName             = "GOOGLE_CLOUD_PROJECT"
)

type SpotZeroFbodService struct {
	loggerProvider     logger.Provider
	BqService          doitBqIface.IfcInserter
	FsService          doitFs.IFbodFirestore
	AwsServiceProvider providers.IAwsServiceProvider
}

func NewSpotScalingFbodService(loggerProvider logger.Provider, conn *connection.Connection) *SpotZeroFbodService {
	ctx := context.Background()
	projectID := os.Getenv(EnvVarName)

	fs := doitFs.NewFbodFirestoreWithClient(conn.Firestore(ctx))

	bqClient, err := doitBq.NewService(ctx, projectID)
	if err != nil {
		panic(err)
	}

	inserter := bqClient.GetInserter("fbod", "fbod-events")

	awsProvider := providers.NewAwsEmptyServiceProvider()

	return &SpotZeroFbodService{
		loggerProvider:     loggerProvider,
		BqService:          inserter,
		FsService:          fs,
		AwsServiceProvider: awsProvider,
	}
}

func (s *SpotZeroFbodService) FbodHealthCheck(ctx context.Context) error {
	log := s.loggerProvider(ctx)

	fbodAsgs, err := s.FsService.GetListOfFbodNotRecoveredAsgs(ctx)
	if err != nil {
		log.Errorf("FbodHealthCheck, error getting fbod list: %s", err)
		return err
	}

	if len(fbodAsgs) == 0 {
		return nil
	}

	doitSession, err := getDoitSession(ctx)
	if err != nil {
		log.Errorf("FbodHealthCheck, error getting doit session: %s", err)
		return err
	}

	common.RunConcurrentJobsOnCollection(ctx, fbodAsgs, 5, func(ctx context.Context, fbodAsgSnap *firestore.DocumentSnapshot) {
		var fbodAsg fsPkg.FbodStatusFsDoc
		if err := fbodAsgSnap.DataTo(&fbodAsg); err != nil {
			log.Errorf("FbodHealthCheck, error parsing fbod doc. err: %s", err)
			return
		}

		s.SingleFbodCheck(ctx, &fbodAsg, doitSession)
	})

	return nil
}

func (s *SpotZeroFbodService) SingleFbodCheck(ctx context.Context, fbodAsg *fsPkg.FbodStatusFsDoc, doitSession *session.Session) {
	log := s.loggerProvider(ctx)

	var eventStatus fsPkg.FbodStatus

	asg, awsSvc, err := s.GetAsgAndAwsSvc(ctx, doitSession, fbodAsg.Event.Account, fbodAsg.Event.Detail.AutoScalingGroupName, fbodAsg.Event.Region)
	if err != nil {
		log.Errorf("SingleFbodCheck: error getting asg and/or awsSvc: %s", err)
		return
	}

	if asg.MixedInstancesPolicy == nil {
		return
	}

	actualCapacity := int64(len(asg.Instances))
	desiredCapacity := *asg.DesiredCapacity
	odBaseCapacity := asg.MixedInstancesPolicy.InstancesDistribution.OnDemandBaseCapacity

	eventStatus.ActualCapacity = &actualCapacity
	eventStatus.DesiredCapacity = &desiredCapacity

	if actualCapacity >= desiredCapacity && *odBaseCapacity != *fbodAsg.OriginalOdBaseCapacity {
		_, err := awsSvc.UpdateAsgMipOdBaseCapacity(asg, *fbodAsg.OriginalOdBaseCapacity)
		if err != nil {
			return
		}

		eventStatus.OdBaseCapacity = fbodAsg.OriginalOdBaseCapacity
		eventStatus.State = fsPkg.FbodStateNormal
		eventStatus.PrevState = fbodAsg.State
		eventStatus.Action = fsPkg.FbodActionRevertOdbc
	} else {
		eventStatus.State = fbodAsg.State
		eventStatus.PrevState = fbodAsg.PrevState
		eventStatus.OdBaseCapacity = odBaseCapacity
		eventStatus.Action = fsPkg.FbodActionOdbcUnchanged
	}

	s.UpdateSingleFbodStatus(ctx, fbodAsg.FbodStatus, eventStatus)
}

func (s *SpotZeroFbodService) UpdateSingleFbodStatus(ctx context.Context, prevEventStatus, eventStatus fsPkg.FbodStatus) {
	log := s.loggerProvider(ctx)
	now := time.Now().UTC()
	event := prevEventStatus.Event
	event.DetailType = "Schedule task watchdog"
	eventStatus.Event = event

	eventStatusFsDoc := fsPkg.FbodStatusFsDoc{
		FbodStatus:  eventStatus,
		TimeUpdated: &now,
	}

	eventStatusBqRecord := fsPkg.EventStatusBqRecord{
		State:                  eventStatus.State,
		PrevState:              prevEventStatus.State,
		Action:                 eventStatus.Action,
		ActualCapacity:         *eventStatus.ActualCapacity,
		DesiredCapacity:        *eventStatus.DesiredCapacity,
		OdBaseCapacity:         *eventStatus.OdBaseCapacity,
		OriginalOdBaseCapacity: *prevEventStatus.OriginalOdBaseCapacity,
		Event:                  *event,
		EventJSON:              "{}",
		Timestamp:              now,
	}

	log.Infof("updating FBOD status from watchdog, account: %s, asg: %s", event.Account, event.Detail.AutoScalingGroupName)

	if err := s.FsService.UpdateFbodStatus(ctx, &eventStatusFsDoc); err != nil {
		log.Errorf("Error updating FBOD status from watchdog, account: %s, asg: %s, err: %s", event.Account, event.Detail.AutoScalingGroupName, err)
	}

	if err := s.BqService.Put(ctx, eventStatusBqRecord); err != nil {
		log.Errorf("Error updating bigquery from watchdog, account: %s, asg: %s, err: %s", event.Account, event.Detail.AutoScalingGroupName, err)
	}
}

func (s *SpotZeroFbodService) GetAsgAndAwsSvc(ctx context.Context, doitSession *session.Session, accountID, asgName, region string) (*autoscaling.Group, doitAwsIface.IAwsService, error) {
	account, err := s.FsService.GetAccountWithAsgOrRegion(ctx, accountID, region, asgName)
	if err != nil {
		return nil, nil, err
	}

	roleArn := sharedAwsPkg.AwsRoleArn{AssumeRoleArn: account.RoleToAssumeArn, RoleExternalID: account.ExternalID}
	awsSvc := s.AwsServiceProvider.GetEmptyService()
	awsSvc.CreateAutoscalingService(doitSession, roleArn, region)

	asg, err := awsSvc.GetASG(asgName)
	if err != nil {
		return nil, nil, err
	}

	return asg, awsSvc, nil
}
func getDoitSession(ctx context.Context) (*session.Session, error) {
	projectID := os.Getenv(EnvVarName)

	secretManagerClient, err := doitSm.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	var doitRoleToAssume sharedAwsPkg.AwsRoleArn
	if err := secretManagerClient.GetSecretContentAsStruct(ctx, projectID, AwsSpot0RoleSecretName, &doitRoleToAssume); err != nil {
		return nil, err
	}

	var awsSpot0AccessKey sharedAwsPkg.AwsAccessKey
	if err := secretManagerClient.GetSecretContentAsStruct(ctx, projectID, Spot0AwsCredSecretName, &awsSpot0AccessKey); err != nil {
		return nil, err
	}

	spot0Session, err := doitAws.GetSessionByCreds(&awsSpot0AccessKey, "us-east-1")
	if err != nil {
		return nil, err
	}

	doitSession, err := doitAws.GetAssumeRoleSession(spot0Session, &doitRoleToAssume, "FBOD-WATCHDOG", "us-east-1")
	if err != nil {
		return nil, err
	}

	return doitSession, nil
}
