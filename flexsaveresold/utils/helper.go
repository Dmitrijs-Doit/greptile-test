package utils

type FlexsaveType string

const (
	ComputeFlexsaveType   FlexsaveType = "compute"
	SageMakerFlexsaveType FlexsaveType = "sagemaker"
	RDSFlexsaveType       FlexsaveType = "rds"

	Resold     = "aws-flexsave-resold"
	Standalone = "aws-flexsave-standalone"

	Active   = "active"
	Pending  = "pending"
	Disabled = "disabled"
)

var FlexsaveTypes = []FlexsaveType{
	ComputeFlexsaveType,
	SageMakerFlexsaveType,
	RDSFlexsaveType,
}

func (fst FlexsaveType) ToTitle() string {
	switch fst {
	case ComputeFlexsaveType:
		return "Compute"
	case SageMakerFlexsaveType:
		return "SageMaker"
	case RDSFlexsaveType:
		return "RDS"
	default:
		return ""
	}
}

func ShouldActivateFlexsave(serviceType FlexsaveType, computeStatus, serviceStatus, payerType string) bool {
	if payerType != Resold {
		return false
	}

	switch serviceType {
	case ComputeFlexsaveType:
		return computeStatus == Pending

	case SageMakerFlexsaveType, RDSFlexsaveType:
		return computeStatus == Active && (serviceStatus == "" || serviceStatus == Pending)

	default:
		return false
	}
}
