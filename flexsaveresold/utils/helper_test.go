package utils

import "testing"

func TestShouldActivateFlexsave(t *testing.T) {
	type args struct {
		serviceType   FlexsaveType
		computeStatus string
		serviceStatus string
		payerType     string
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "activates compute",
			args: args{serviceType: ComputeFlexsaveType, computeStatus: Pending, serviceStatus: Pending, payerType: Resold},
			want: true,
		},
		{
			name: "activates compute",
			args: args{serviceType: ComputeFlexsaveType, computeStatus: Pending, serviceStatus: "", payerType: Resold},
			want: true,
		},
		{
			name: "does not activate compute",
			args: args{serviceType: ComputeFlexsaveType, computeStatus: Active, serviceStatus: "", payerType: Resold},
			want: false,
		},
		{
			name: "does not activate compute",
			args: args{serviceType: ComputeFlexsaveType, computeStatus: Disabled, serviceStatus: "", payerType: Resold},
			want: false,
		},
		{
			name: "activates sagemaker",
			args: args{serviceType: SageMakerFlexsaveType, computeStatus: Active, serviceStatus: Pending, payerType: Resold},
			want: true,
		},
		{
			name: "activates sagemaker",
			args: args{serviceType: SageMakerFlexsaveType, computeStatus: Active, serviceStatus: "", payerType: Resold},
			want: true,
		},
		{
			name: "does not activate sagemaker",
			args: args{serviceType: SageMakerFlexsaveType, computeStatus: Pending, serviceStatus: Pending, payerType: Resold},
			want: false,
		},
		{
			name: "does not activate sagemaker",
			args: args{serviceType: SageMakerFlexsaveType, computeStatus: Active, serviceStatus: Active, payerType: Resold},
			want: false,
		},
		{
			name: "activates rds",
			args: args{serviceType: RDSFlexsaveType, computeStatus: Active, serviceStatus: Pending, payerType: Resold},
			want: true,
		},
		{
			name: "activates rds",
			args: args{serviceType: RDSFlexsaveType, computeStatus: Active, serviceStatus: "", payerType: Resold},
			want: true,
		},
		{
			name: "does not activate rds",
			args: args{serviceType: RDSFlexsaveType, computeStatus: Pending, serviceStatus: Pending, payerType: Resold},
			want: false,
		},
		{
			name: "does not activate rds",
			args: args{serviceType: RDSFlexsaveType, computeStatus: Disabled, serviceStatus: Pending, payerType: Resold},
			want: false,
		},
		{
			name: "does not activate Standalone payer",
			args: args{serviceType: RDSFlexsaveType, computeStatus: Pending, serviceStatus: Pending, payerType: Standalone},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldActivateFlexsave(tt.args.serviceType, tt.args.computeStatus, tt.args.serviceStatus, tt.args.payerType); got != tt.want {
				t.Errorf("ShouldActivateFlexsave() = %v, want %v", got, tt.want)
			}
		})
	}
}
