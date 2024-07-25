package doitproducts

import "testing"

func Test_getFixedSkuID(t *testing.T) {
	type args struct {
		packageType string
		packageName string
		paymentTerm string
		pointOfSale string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr *string
	}{
		{
			name:    "navigator sku",
			args:    args{"navigator", "standard", "monthly", "default"},
			want:    "P-ST-M-D-001",
			wantErr: nil,
		},
		{
			name:    "navigator sku",
			args:    args{"navigator", "standard", "monthly", "aws-marketplace"},
			want:    "P-ST-M-R-001",
			wantErr: nil,
		},
		{
			name:    "navigator sku",
			args:    args{"navigator", "enhanced", "monthly", "aws-marketplace"},
			want:    "P-ET-M-R-001",
			wantErr: nil,
		},
		{
			name:    "navigator sku",
			args:    args{"navigator", "enhanced", "annual", "aws-marketplace"},
			want:    "P-ET-A-R-001",
			wantErr: nil,
		},
		{
			name:    "navigator sku",
			args:    args{"navigator", "premium", "monthly", "aws-marketplace"},
			want:    "P-PT-M-R-001",
			wantErr: nil,
		},
		{
			name:    "navigator sku",
			args:    args{"navigator", "enterprise", "monthly", "aws-marketplace"},
			want:    "P-EP-M-R-001",
			wantErr: nil,
		},
		{
			name:    "solve sku",
			args:    args{"solve", "premium", "monthly", "aws-marketplace"},
			want:    "S-PT-M-R-001",
			wantErr: nil,
		},
		{
			name:    "solve sku",
			args:    args{"solve", "enterprise", "annual", "aws-marketplace"},
			want:    "S-EP-A-R-001",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getFixedSkuID(tt.args.packageType, tt.args.packageName, tt.args.paymentTerm, tt.args.pointOfSale)
			if tt.wantErr != nil {
				if (err == nil) || (err != nil && *tt.wantErr != err.Error()) {
					t.Errorf("getFixedSkuID() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			if got != tt.want {
				t.Errorf("getFixedSkuID() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getVariableSkuID(t *testing.T) {
	type args struct {
		packageType string
		packageName string
		paymentTerm string
		pointOfSale string
		cloud       string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr *string
	}{
		{
			name:    "solve sku",
			args:    args{"solve", "standard", "monthly", "default", "google-cloud-platform"},
			want:    "S-ST-M-D-002",
			wantErr: nil,
		},
		{
			name:    "solve sku",
			args:    args{"solve", "standard", "annual", "default", "amazon-web-services"},
			want:    "S-ST-A-D-003",
			wantErr: nil,
		},
		{
			name:    "solve sku",
			args:    args{"solve", "standard", "monthly", "aws-marketplace", "azure"},
			want:    "S-ST-M-R-004",
			wantErr: nil,
		},
		{
			name:    "solve sku",
			args:    args{"solve", "standard", "monthly", "default", "looker"},
			want:    "S-ST-M-D-005",
			wantErr: nil,
		},
		{
			name:    "solve sku",
			args:    args{"solve", "standard", "monthly", "aws-marketplace", "office-365"},
			want:    "S-ST-M-R-006",
			wantErr: nil,
		},
		{
			name:    "solve sku",
			args:    args{"solve", "standard", "monthly", "default", "google-workplace"},
			want:    "S-ST-M-D-007",
			wantErr: nil,
		},
		{
			name:    "solve sku",
			args:    args{"solve", "enterprise", "monthly", "default", "google-workplace"},
			want:    "S-EP-M-D-007",
			wantErr: nil,
		},
		{
			name:    "solve sku",
			args:    args{"solve", "premium", "monthly", "default", "google-workplace"},
			want:    "S-PT-M-D-007",
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getVariableSkuID(tt.args.packageType, tt.args.packageName, tt.args.paymentTerm, tt.args.pointOfSale, tt.args.cloud)
			if tt.wantErr != nil {
				if (err == nil) || (err != nil && *tt.wantErr != err.Error()) {
					t.Errorf("getFixedSkuID() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}
			if got != tt.want {
				t.Errorf("getFixedSkuID() got = %v, want %v", got, tt.want)
			}
		})
	}
}
