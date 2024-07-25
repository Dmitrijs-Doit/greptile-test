package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/sfn"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/spot0/api/model"
)

const (
	spot0DevStateMachineARNusWest1          = "arn:aws:states:us-west-1:535390606960:stateMachine:spotzeroStateMachine"
	spot0DevLambdaManageAsgEventsRulesWest1 = "arn:aws:lambda:us-west-1:535390606960:function:spot0-ManageAsgEventsRules"

	spot0DevStateMachineARNusEast1          = "arn:aws:states:us-east-1:535390606960:stateMachine:spotzeroStateMachine"
	spot0DevLambdaManageAsgEventsRulesEast1 = "arn:aws:lambda:us-east-1:535390606960:function:spot0-ManageAsgEventsRules"
	spot0DevLambdaUpdateAsgConfig           = "arn:aws:lambda:us-east-1:535390606960:function:spot0-UpdateAsgConfig"
	spot0DevAveragePrices                   = "arn:aws:lambda:us-east-1:535390606960:function:spot0-CalculateCost"

	spot0ProdStateMachineARNusEast1          = "arn:aws:states:us-east-1:684965592724:stateMachine:spotzeroStateMachine"
	spot0ProdLambdaManageAsgEventsRulesEast1 = "arn:aws:lambda:us-east-1:684965592724:function:spot0-ManageAsgEventsRules"
	spot0ProdLambdaUpdateAsgConfig           = "arn:aws:lambda:us-east-1:684965592724:function:spot0-UpdateAsgConfig"
	spot0ProdAveragePrices                   = "arn:aws:lambda:us-east-1:684965592724:function:spot0-CalculateCost"

	asgScope = "asg"

	lambdaInvokeError = "could not invoke lambda. error %s"
)

var (
	ErrForbidden           = errors.New("Forbidden")
	ErrInternalServerError = errors.New("Internal Server Error")
	ErrNotFound            = errors.New("Not Found")
)

type SpotZeroService struct {
	*logger.Logging
	*connection.Connection
	stepFunctionsAPI                 *sfn.SFN
	lambdaAPI                        *lambda.Lambda
	spotZeroStateMachine             string
	manageAsgEventsRulesStateMachine string
	updateAsgConfigLambda            string
	averagePricesLambda              string
}
type SpotScalingDemoService struct {
	*logger.Logging
	*connection.Connection
}

func NewSpotScalingDemoService(log *logger.Logging, conn *connection.Connection) *SpotScalingDemoService {
	return &SpotScalingDemoService{
		log,
		conn,
	}
}

type SpotScalingServiceInterface interface {
	ExecuteSpotScaling(ctx context.Context, req *model.ApplyConfigurationRequest) (*model.Response, error)
	AveragePrices(ctx context.Context, req *model.AveragePricesRequest) (*model.AveragePricesResponse, error)
	UpdateAsgConfig(ctx context.Context, req *model.UpdateAsgConfigRequest) (*model.Response, error)
	UpdateFallbackOnDemandConfig(ctx context.Context, req *model.FallbackOnDemandRequest) (*model.Response, error)
}

func NewSpotZeroService(log *logger.Logging, conn *connection.Connection) *SpotZeroService {
	s := initAWSSession(log)

	spotZeroStateMachine := spot0DevStateMachineARNusEast1
	manageAsgEventsRulesStateMachine := spot0DevLambdaManageAsgEventsRulesEast1
	updateAsgConfigLambda := spot0DevLambdaUpdateAsgConfig
	averagePricesLambda := spot0DevAveragePrices

	if common.Env == "production" {
		spotZeroStateMachine = spot0ProdStateMachineARNusEast1
		manageAsgEventsRulesStateMachine = spot0ProdLambdaManageAsgEventsRulesEast1
		updateAsgConfigLambda = spot0ProdLambdaUpdateAsgConfig
		averagePricesLambda = spot0ProdAveragePrices
	}

	return &SpotZeroService{
		log,
		conn,
		sfn.New(s),
		lambda.New(s),
		spotZeroStateMachine,
		manageAsgEventsRulesStateMachine,
		updateAsgConfigLambda,
		averagePricesLambda,
	}
}

func (s *SpotZeroService) UpdateAsgConfig(ctx context.Context, req *model.UpdateAsgConfigRequest) (*model.Response, error) {
	logger := s.Logger(ctx)

	epErrMsg := "could not update asg config"
	epErr := fmt.Errorf(epErrMsg)

	// execute lambda
	buf, err := json.Marshal(req)
	if err != nil {
		logger.Errorf("could not marshal asg config event. error %s", err)
		return &model.Response{Done: false, ErrorMessage: epErrMsg}, epErr
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(s.updateAsgConfigLambda),
		Payload:      buf,
	}

	result, err := s.lambdaAPI.Invoke(input)
	if err != nil {
		logger.Errorf(lambdaInvokeError, err)
		return &model.Response{Done: false, ErrorMessage: epErrMsg}, epErr
	}

	var resp model.UpdateAsgConfigResponse

	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		logger.Errorf("could not unmarshal lambda response. error %s", err)
		return &model.Response{Done: false, ErrorMessage: epErrMsg}, epErr
	}

	if !resp.Success {
		return &model.Response{
			Done: resp.Success,
		}, &model.UpdateAsgConfigError{Code: resp.ErrorCode, Err: errors.New(resp.ErrorDesc)}
	}

	// Refresh ASG in normal flow
	refreshRequest := &model.ApplyConfigurationRequest{
		Scope:      asgScope,
		CustomerID: req.CustomerID,
		Region:     req.Region,
		ASGName:    req.AsgName,
		AccountID:  req.AccountID,
	}

	res, err := s.ExecuteSpotScaling(ctx, refreshRequest)
	if err != nil {
		logger.Errorf("could not refresh asg. error %s", err)
		return &model.Response{Done: false, ErrorMessage: epErrMsg}, epErr
	}

	if !res.Done {
		logger.Errorf("could not refresh asg. error %s", res.ErrorMessage)
		return &model.Response{Done: false, ErrorMessage: epErrMsg}, epErr
	}

	return &model.Response{
		Done: true,
	}, nil
}

func (s *SpotZeroService) AveragePrices(ctx context.Context, req *model.AveragePricesRequest) (*model.AveragePricesResponse, error) {
	logger := s.Logger(ctx)

	epErr := fmt.Errorf("could not execute average prices")

	// execute lambda
	buf, err := json.Marshal(req)
	if err != nil {
		logger.Errorf("could not marshal request. error %s", err)
		return &model.AveragePricesResponse{}, epErr
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(s.averagePricesLambda),
		Payload:      buf,
	}

	result, err := s.lambdaAPI.Invoke(input)
	if err != nil {
		logger.Errorf(lambdaInvokeError, err)
		return &model.AveragePricesResponse{}, epErr
	}

	var resp model.AveragePricesLambdaResponse

	err = json.Unmarshal(result.Payload, &resp)
	if err != nil {
		logger.Errorf("could not unmarshal lambda response. error %s", err)
		return &model.AveragePricesResponse{}, epErr
	}

	if !resp.Success {
		logger.Error("lambda failed")
		return &model.AveragePricesResponse{}, epErr
	}

	return &model.AveragePricesResponse{
		SpotHourCost:     resp.SpotHourCost,
		OnDemandHourCost: resp.OnDemandHourCost,
	}, nil
}

func (s *SpotZeroService) UpdateFallbackOnDemandConfig(ctx context.Context, req *model.FallbackOnDemandRequest) (*model.Response, error) {
	err := s.getAccountDetails(ctx, req)
	if err != nil {
		return &model.Response{
			Done:         false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// execute lambda
	err = s.ExecuteFallbackOnDemand(ctx, req)
	if err != nil {
		return &model.Response{
			Done:         false,
			ErrorMessage: err.Error(),
		}, nil
	}

	return &model.Response{
		Done: true,
	}, nil
}

func initAWSSession(log *logger.Logging) *session.Session {
	logger := log.Logger(context.Background())

	creds, err := secretmanager.AccessSecretLatestVersion(context.Background(), secretmanager.SecretSpot0AwsCred)
	if err != nil || creds == nil {
		logger.Errorf("could not get spot zero aws creds. error %s", err)
		return nil
	}

	var awsCred AwsCred

	err = json.Unmarshal(creds, &awsCred)
	if err != nil {
		logger.Errorf("could not unmarshal spot zero aws creds. error %s", err)
		return nil
	}

	s, err := session.NewSession(&aws.Config{
		Region:      aws.String("us-east-1"),
		Credentials: credentials.NewStaticCredentials(awsCred.AwsAccessKey, awsCred.AwsSecretAccessKey, ""),
		MaxRetries:  aws.Int(2),
	})
	if err != nil {
		logger.Errorf("could not create new spot zero aws session. error %s", err)
		return nil
	}

	return s
}

func (s *SpotZeroService) startAWSStepFunction(ctx context.Context, evt *GetAccountsEvent) (*string, error) {
	logger := s.Logger(ctx)

	buf, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("could not marshal account event. error %s", err)
	}

	startExecutionInput := sfn.StartExecutionInput{
		StateMachineArn: aws.String(s.spotZeroStateMachine),
		Input:           aws.String(string(buf)),
		Name:            &evt.ExecID,
	}

	startExecutionOutput, err := s.stepFunctionsAPI.StartExecution(&startExecutionInput)
	if err != nil {
		logger.Error(err)
		return nil, fmt.Errorf("could not start execution of aws step function. error %s", err)
	}

	if startExecutionOutput == nil {
		err := errors.New("could not proceed. retrieved an empty output from aws step function")
		logger.Error(err)

		return nil, err
	}

	return startExecutionOutput.ExecutionArn, nil
}

func (s *SpotZeroService) invokeLambda(ctx context.Context, evt *model.FallbackOnDemandRequest) error {
	l := s.Logger(ctx)

	buf, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("could not marshal account event. error %s", err)
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(s.manageAsgEventsRulesStateMachine),
		Payload:      buf,
	}

	result, err := s.lambdaAPI.Invoke(input)
	if err != nil {
		l.Errorf(lambdaInvokeError, err)
		return err
	}

	if result.FunctionError != nil {
		l.Info(string(result.Payload))

		type Payload struct {
			ErrorMessage string `json:"errorMessage"`
		}

		var payload Payload

		err = json.Unmarshal(result.Payload, &payload)
		if err != nil {
			l.Errorf("could not unmarshal payload. error %s", err)
			return err
		}

		return errors.New(payload.ErrorMessage)
	}

	return nil
}

func (s *SpotZeroService) verifyCustomer(ctx context.Context, customerID string) error {
	logger := s.Logger(ctx)

	customerRef := s.Firestore(ctx).Collection("customers").Doc(customerID)
	query := customerRef.Collection("cloudConnect")

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		err := fmt.Errorf("could not retrieve docs from firestore. error %s", err)
		logger.Error(err)

		return err
	}

	if len(docSnaps) == 0 {
		err := fmt.Errorf("could not proceed. there are no accounts related to the customer")
		logger.Error(err)

		return err
	}

	return nil
}
