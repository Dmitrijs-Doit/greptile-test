package costs

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"

	doitAwsIface "github.com/doitintl/aws/iface"
	sharedAwsPkg "github.com/doitintl/aws/pkg"
	"github.com/doitintl/aws/providers"
	mpaFs "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	bq "github.com/doitintl/hello/scheduled-tasks/spot0/dal/bigquery"
	sl "github.com/doitintl/hello/scheduled-tasks/spot0/dal/slack"
	doitSm "github.com/doitintl/secretmanager"
	iDoitSm "github.com/doitintl/secretmanager/iface"
)

const (
	AwsSpot0RoleSecretName = "aws-spot0-role"
	Spot0AwsCredSecretName = "spot0-aws-cred"
	EnvVarName             = "GOOGLE_CLOUD_PROJECT"
)

type SpotZeroCostsExplorerService struct {
	loggerProvider     logger.Provider
	bqService          bq.ISpot0CostsBigQuery
	mpaFsService       mpaFs.MasterPayerAccounts
	smClient           iDoitSm.ISecretClient
	awsServiceProvider providers.ICostExplorerProvider
	slackService       sl.ISlack
}

func NewSpotZeroCostsExplorerService(loggerProvider logger.Provider, conn *connection.Connection) *SpotZeroCostsExplorerService {
	ctx := context.Background()

	bqService, err := bq.NewBigQueryService(ctx)
	if err != nil {
		panic(err)
	}

	mpaFsService := mpaFs.NewMasterPayerAccountDALWithClient(conn.Firestore(ctx))

	smClient, err := doitSm.NewClient(ctx)
	if err != nil {
		panic(err)
	}

	awsCeProvider := providers.NewCostExplorerProvider()

	var slackService sl.Spot0Slack

	return &SpotZeroCostsExplorerService{
		loggerProvider,
		bqService,
		mpaFsService,
		smClient,
		awsCeProvider,
		&slackService,
	}
}

func (s *SpotZeroCostsExplorerService) UpdateCostAllocationTags(ctx context.Context) error {
	log := s.loggerProvider(ctx)

	nonBillingDomains, err := s.bqService.GetNonBillingTagsDomains(ctx)
	if err != nil {
		return err
	}

	awsService, doitSession, err := s.getAwsServiceAndSession(ctx)
	if err != nil {
		log.Error("updateCostAllocationTags: error getting aws service and/or session")
		return err
	}

	var reportData []*sl.NonBillingTagsRow

	for _, nonBillingDomain := range nonBillingDomains {
		masterPayerAccounts, err := s.mpaFsService.GetMasterPayerAccountsForDomain(ctx, nonBillingDomain.PrimaryDomain)
		if err != nil {
			log.Errorf("UpdateCostAllocationTags: can't get mpa for domain %s: ", nonBillingDomain.PrimaryDomain)
			continue
		}

		for _, mpa := range masterPayerAccounts {
			if mpa.Features.NRA {
				log.Infof("UpdateCostAllocationTags: Domain %s, account %s has no root access", mpa.Domain, mpa.AccountNumber)

				reportData = append(reportData, &sl.NonBillingTagsRow{
					PrimaryDomain: mpa.Domain,
					MPAAccount:    mpa.AccountNumber,
					Status:        "NRA",
				})

				continue
			}

			if err := s.UpdateCostAllocationSingleAccountTags(awsService, doitSession, mpa); err != nil {
				log.Errorf("UpdateCostAllocationTags: error updating cost allocation tags for domain %s, account %s  ", mpa.Domain, mpa.AccountNumber, err)

				reportData = append(reportData, &sl.NonBillingTagsRow{
					PrimaryDomain: mpa.Domain,
					MPAAccount:    mpa.AccountNumber,
					Status:        err.Error(),
				})

				continue
			}

			reportData = append(reportData, &sl.NonBillingTagsRow{
				PrimaryDomain: mpa.Domain,
				MPAAccount:    mpa.AccountNumber,
				Status:        "Updated",
			})

			log.Infof("UpdateCostAllocationTags: Domain %s, account %s updated", mpa.Domain, mpa.AccountNumber)
		}
	}

	if _, err := s.slackService.PublishToSlack(ctx, reportData); err != nil {
		log.Errorf("UpdateCostAllocationTags: error publishing to slack, ", err)
	}

	return nil
}

func (s *SpotZeroCostsExplorerService) UpdateCostAllocationSingleAccountTags(awsService doitAwsIface.ICostExplorerService, doitSession *session.Session, mpa *domain.MasterPayerAccount) error {
	name := "aws:autoscaling:groupName"
	statusActive := costexplorer.CostAllocationTagStatusActive

	statusEntry := costexplorer.CostAllocationTagStatusEntry{
		Status: &statusActive,
		TagKey: &name,
	}

	updateCostAllocationTagsStatusInput := costexplorer.UpdateCostAllocationTagsStatusInput{
		CostAllocationTagsStatus: []*costexplorer.CostAllocationTagStatusEntry{&statusEntry},
	}

	// for dev purposes (use with spot0Session)
	//roleToAssumeArn := "arn:aws:iam::459917102817:role/spot0-cost-allocation-tags-development"
	roleArn := sharedAwsPkg.AwsRoleArn{
		AssumeRoleArn:  mpa.RoleARN,
		RoleExternalID: *mpa.CustomerID,
	}
	awsService.CreateCostExplorerService(doitSession, roleArn)

	if _, err := awsService.UpdateCostAllocationTagsStatus(&updateCostAllocationTagsStatusInput); err != nil {
		return err
	}

	return nil
}

func (s *SpotZeroCostsExplorerService) getServiceSecrets(ctx context.Context, projectID string) (*sharedAwsPkg.AwsRoleArn, *sharedAwsPkg.AwsAccessKey, error) {
	var doitRoleToAssume sharedAwsPkg.AwsRoleArn
	if err := s.smClient.GetSecretContentAsStruct(ctx, projectID, AwsSpot0RoleSecretName, &doitRoleToAssume); err != nil {
		return nil, nil, err
	}

	var awsSpot0AccessKey sharedAwsPkg.AwsAccessKey
	if err := s.smClient.GetSecretContentAsStruct(ctx, projectID, Spot0AwsCredSecretName, &awsSpot0AccessKey); err != nil {
		return nil, nil, err
	}

	return &doitRoleToAssume, &awsSpot0AccessKey, nil
}

func (s *SpotZeroCostsExplorerService) getAwsServiceAndSession(ctx context.Context) (doitAwsIface.ICostExplorerService, *session.Session, error) {
	projectID := os.Getenv(EnvVarName)

	doitRoleToAssume, awsSpot0AccessKey, err := s.getServiceSecrets(ctx, projectID)
	if err != nil {
		return nil, nil, err
	}

	awsService := s.awsServiceProvider.GetEmptyService()

	spot0Session, err := awsService.GetSessionByCreds(awsSpot0AccessKey, "us-east-1")
	if err != nil {
		return nil, nil, err
	}

	doitSession, err := awsService.GetAssumeRoleSession(spot0Session, doitRoleToAssume, "SpotScalingListCostAllocationTags", "us-east-1")
	if err != nil {
		return nil, nil, err
	}

	return awsService, doitSession, nil
}
