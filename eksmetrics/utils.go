package eksmetrics

import (
	"context"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type EKSMetricsRequest struct {
	EKSClusterID string `json:"external_id"`
	Arn          string `json:"management_arn"`
	AccountID    string `json:"account_id"`
	StackID      string `json:"stack_id"`
	StackName    string `json:"stack_name"`
	CURPath      string `json:"cur_path"`
	S3Bucket     string `json:"s3_bucket"`
	Region       string `json:"-,omitempty"`
}

const (
	devSlackChannel  = "#eks-ops-dev"
	prodSlackChannel = "#eks-ops-prod"
)

var (
	// errors
	ErrorEKSClusterID = errors.New("missing EKS cluster id")
	ErrorAccountID    = errors.New("missing AWS account id")
	ErrorRegion       = errors.New("missing Region")
	ErrorStackID      = errors.New("missing EKS stack id")
)

func (s *EKSMetricsService) getLogger(ctx context.Context, customerID string) logger.ILogger {
	l := s.loggerProvider(ctx)

	l.SetLabels(map[string]string{
		logger.LabelCustomerID: customerID,
		"service":              "eks-metrics",
		"flow":                 "onboarding",
	})

	return l
}

func (s *EKSMetricsService) ParseRequest(ctx *gin.Context) *EKSMetricsRequest {
	var req EKSMetricsRequest
	logger := s.getLogger(ctx, req.EKSClusterID)

	if err := ctx.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		logger.Errorf("Error parse the CF %v, %v", req, err)
	}

	req.Region = ExtractRegionFromStackID(req.StackID)

	return &req
}

func ExtractRegionFromStackID(str string) string {
	arn := strings.Split(str, "/")

	var region []string
	if len(arn) > 0 {
		region = strings.Split(arn[0], ":")
		if len(region) > 1 {
			return region[3]
		}
	}

	return ""
}

// validateFields validates mandatory fields has received from the request (fullPayload true for activation endpoint)
func (s *EKSMetricsService) validateFields(req *EKSMetricsRequest) error {
	if len(req.StackID) == 0 {
		return ErrorStackID
	}

	if len(req.EKSClusterID) == 0 {
		return ErrorEKSClusterID
	}

	if len(req.AccountID) == 0 {
		return ErrorAccountID
	}

	if len(req.Region) == 0 {
		return ErrorRegion
	}

	return nil
}

func (s *EKSMetricsService) UpdateEksStackCreationStatus(ctx context.Context, req *EKSMetricsRequest, status string) error {
	logger := s.getLogger(ctx, req.EKSClusterID)

	logger.Infof("Update eks started %s: %s, %s, %s\n", status, req.AccountID, req.Region, req.EKSClusterID)

	if err := s.validateFields(req); err != nil {
		logger.Errorf("Error validating the CF fields %v, %v", req, err)

		return err
	}

	if resp, err := s.SetCloudformationStatus(ctx, req.AccountID, req.Region, req.EKSClusterID, status); err != nil || resp.StatusCode != 200 {
		logger.Errorf("Error updating the CF status %v, %v", req, err)

		return err
	}

	logger.Infof("Update eks success %s, %s, %s\n", req.AccountID, req.Region, req.EKSClusterID)

	return nil
}
