package domain

import (
	"testing"
)

func TestProduct_ExtractProduct(t *testing.T) {
	type args struct {
		productName string
	}

	tests := []struct {
		name        string
		args        args
		wantErr     bool
		err         error
		expectedRes Product
	}{
		{
			name: "extract valid product",
			args: args{
				productName: "doit-flexsave-development.endpoints.doit-intl-public.cloud.goog",
			},
			wantErr:     false,
			expectedRes: ProductFlexsaveDevelopment,
		},
		{
			name: "error on random string with dot separator",
			args: args{
				productName: "aaaaa.bbb",
			},
			wantErr: true,
		},
		{
			name: "error on random string without dot",
			args: args{
				productName: "aaaaa",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractProduct(tt.args.productName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractProduct() error = %v, wantErr %v", err, tt.wantErr)
			}

			if (err != nil) && result != tt.expectedRes {
				t.Errorf("ExtractProduct() result = %v, expectedRes %v", result, tt.expectedRes)
			}
		})
	}
}
