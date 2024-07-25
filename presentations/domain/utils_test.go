package domain

import "testing"

func TestHashCustomerIdIntoABillingAccountId(t *testing.T) {
	type args struct {
		customerID1 string
		customerID2 string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Check that the same id hashes to the same billing account id",
			args: args{
				customerID1: "123",
				customerID2: "123",
			},
			want: true,
		},
		{
			name: "Check that different ids hash to different billing account ids",
			args: args{
				customerID1: "123",
				customerID2: "456",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got1 := HashCustomerIdIntoABillingAccountId(tt.args.customerID1)
			got2 := HashCustomerIdIntoABillingAccountId(tt.args.customerID2)
			equal := got1 == got2
			if equal != tt.want {
				t.Errorf("HashCustomerIdIntoABillingAccountId1() = %v, HashCustomerIdIntoABillingAccountId2() = %v, want %v", got1, got2, tt.want)
			}
		})
	}
}

func TestHashCustomerIdIntoABillingAccountId2(t *testing.T) {
	tests := []struct {
		name string
		in   string
		out  string
	}{
		{
			name: "should generate proper billing account ID format",
			in:   "presentationcustomerAWSAzureGCP",
			out:  "9BE1A5-1A5468-FCD372",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := HashCustomerIdIntoABillingAccountId(tt.in)
			if got != tt.out {
				t.Errorf("for %v, expected %v, but got %v", tt.in, tt.out, got)
			}
		})
	}
}
