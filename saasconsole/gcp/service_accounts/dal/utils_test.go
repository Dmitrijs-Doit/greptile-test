package dal

import (
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	ds "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dataStructures"
)

const (
	projectID                   string = "doitintl-sa-fs-p1"
	serviceAccountEmailTemplate string = "sa-fs-e%d@%s.iam.gserviceaccount.com"
)

func createNServiceAccounts(n int) []ds.ServiceAccountMetadata {
	pool := make([]ds.ServiceAccountMetadata, 0)
	for i := 1; i <= n; i++ {
		pool = append(pool, createServiceAccount(i))
	}

	return pool
}

func createServiceAccount(args ...interface{}) ds.ServiceAccountMetadata {
	metadata := ds.ServiceAccountMetadata{}

	for i, a := range args {
		switch i {
		case 0:
			metadata.ServiceAccountEmail = getServiceAccountEmail(a.(int))
		case 1:
			metadata.BillingAccountID = a.(string)
		case 2:
			metadata.ProjectID = a.(string)
		}
	}

	return metadata
}

func getServiceAccountEmail(i int) string {
	return fmt.Sprintf(serviceAccountEmailTemplate, i, projectID)
}

func getCustomerRef() *firestore.DocumentRef {
	return &firestore.DocumentRef{ID: "123456789"}
}

func Test_addNewServiceAccount(t *testing.T) {
	type args struct {
		pool                *ds.FreeServiceAccountsPool
		serviceAccountEmail string
	}

	tests := []struct {
		name  string
		args  args
		want  int
		want1 *ds.FreeServiceAccountsPool
	}{
		// TODO: Add test cases.
		{
			name: "add new service account",
			args: args{
				pool: &ds.FreeServiceAccountsPool{
					FreeServiceAccounts: createNServiceAccounts(2),
				},
				serviceAccountEmail: getServiceAccountEmail(3),
			},
			want: 3,
			want1: &ds.FreeServiceAccountsPool{
				FreeServiceAccounts: createNServiceAccounts(3),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addNewServiceAccount(tt.args.pool, tt.args.serviceAccountEmail)
		})

		got := len(tt.args.pool.FreeServiceAccounts)
		if got != tt.want {
			t.Errorf("addNewServiceAccount() got = %v, want %v", got, tt.want)
		}

		for i, sa := range tt.args.pool.FreeServiceAccounts {
			if sa.ServiceAccountEmail != tt.want1.FreeServiceAccounts[i].ServiceAccountEmail {
				t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", tt.args.pool.FreeServiceAccounts, tt.want1.FreeServiceAccounts)
			}
		}
	}
}

func Test_getDedicatedServiceAccountEmail(t *testing.T) {
	type args struct {
		pool             *ds.FreeServiceAccountsPool
		reserved         *ds.CustomerMetadata
		customerRef      *firestore.DocumentRef
		billingAccountID string
	}

	tests := []struct {
		name    string
		args    args
		want    string
		want1   bool
		want2   *ds.FreeServiceAccountsPool
		want3   *ds.CustomerMetadata
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "return already reserved service account email",
			args: args{
				pool: &ds.FreeServiceAccountsPool{
					FreeServiceAccounts: createNServiceAccounts(2),
				},
				reserved: &ds.CustomerMetadata{
					Customer: getCustomerRef(),
					ServiceAccounts: []ds.ServiceAccountMetadata{
						createServiceAccount(3),
					},
				},
				customerRef: getCustomerRef(),
			},
			want:  getServiceAccountEmail(3),
			want1: false,
			want2: &ds.FreeServiceAccountsPool{
				FreeServiceAccounts: createNServiceAccounts(2),
			},
			want3: &ds.CustomerMetadata{
				Customer: getCustomerRef(),
				ServiceAccounts: []ds.ServiceAccountMetadata{
					createServiceAccount(3),
				},
			},
		},
		{
			name: "empty free accounts pool",
			args: args{
				pool: &ds.FreeServiceAccountsPool{
					FreeServiceAccounts: []ds.ServiceAccountMetadata{},
				},
				reserved:    &ds.CustomerMetadata{},
				customerRef: getCustomerRef(),
			},
			wantErr: true,
		},
		{
			name: "reserve service account for customer",
			args: args{
				pool: &ds.FreeServiceAccountsPool{
					FreeServiceAccounts: createNServiceAccounts(2),
				},
				reserved:    &ds.CustomerMetadata{},
				customerRef: getCustomerRef(),
			},
			want:  getServiceAccountEmail(1),
			want1: true,
			want2: &ds.FreeServiceAccountsPool{
				FreeServiceAccounts: []ds.ServiceAccountMetadata{
					createServiceAccount(2),
				},
			},
			want3: &ds.CustomerMetadata{
				Customer: getCustomerRef(),
				ServiceAccounts: []ds.ServiceAccountMetadata{
					createServiceAccount(1),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := getDedicatedServiceAccountEmail(tt.args.pool, tt.args.reserved, &ds.CustomerMetadata{}, tt.args.customerRef, tt.args.billingAccountID)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("getReservedServiceAccountEmail() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if got != tt.want {
				t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", got, tt.want)
			}

			if got1 != tt.want1 {
				t.Errorf("getReservedServiceAccountEmail() got1 = %v, want %v", got1, tt.want1)
			}

			if len(tt.args.pool.FreeServiceAccounts) != len(tt.want2.FreeServiceAccounts) {
				t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", tt.args.pool.FreeServiceAccounts, tt.want2.FreeServiceAccounts)
			}

			for i, sa := range tt.args.pool.FreeServiceAccounts {
				if sa.ServiceAccountEmail != tt.want2.FreeServiceAccounts[i].ServiceAccountEmail {
					t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", tt.args.pool.FreeServiceAccounts, tt.want2.FreeServiceAccounts)
					break
				}
			}

			if len(tt.args.reserved.ServiceAccounts) != len(tt.want3.ServiceAccounts) {
				t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", tt.args.reserved.ServiceAccounts, tt.want3.ServiceAccounts)
			}

			for i, sa := range tt.args.reserved.ServiceAccounts {
				if sa.ServiceAccountEmail != tt.want3.ServiceAccounts[i].ServiceAccountEmail {
					t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", tt.args.reserved.ServiceAccounts, tt.want3.ServiceAccounts)
					break
				}
			}

			if tt.args.reserved.Customer == nil {
				t.Errorf("getReservedServiceAccountEmail() got1 = %v, want %v", "nil pointer", tt.args.customerRef.ID)
			}

			if tt.args.reserved.Customer != nil && tt.args.reserved.Customer.ID != tt.want3.Customer.ID {
				t.Errorf("getReservedServiceAccountEmail() got1 = %v, want %v", tt.args.customerRef, tt.want3.Customer)
			}
		})
	}
}

func Test_acquireServiceAccount(t *testing.T) {
	type args struct {
		reserved            *ds.CustomerMetadata
		acquired            *ds.CustomerMetadata
		serviceAccountEmail string
		billingAccountID    string
	}

	tests := []struct {
		name    string
		args    args
		want1   *ds.CustomerMetadata
		want2   *ds.CustomerMetadata
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "no reseerved service account",
			args: args{
				reserved: &ds.CustomerMetadata{
					Customer:        getCustomerRef(),
					ServiceAccounts: createNServiceAccounts(3),
				},
				acquired:            &ds.CustomerMetadata{},
				serviceAccountEmail: getServiceAccountEmail(5),
				billingAccountID:    "0000-1111-2222-3333",
			},
			want1: &ds.CustomerMetadata{
				Customer:        getCustomerRef(),
				ServiceAccounts: createNServiceAccounts(3),
			},
			want2:   &ds.CustomerMetadata{},
			wantErr: true,
		},
		{
			name: "acquire service account",
			args: args{
				reserved: &ds.CustomerMetadata{
					Customer:        getCustomerRef(),
					ServiceAccounts: createNServiceAccounts(3),
				},
				acquired:            &ds.CustomerMetadata{},
				serviceAccountEmail: getServiceAccountEmail(2),
				billingAccountID:    "0000-1111-2222-3333",
			},
			want1: &ds.CustomerMetadata{
				Customer: getCustomerRef(),
				ServiceAccounts: []ds.ServiceAccountMetadata{
					createServiceAccount(1),
					createServiceAccount(3),
				},
			},
			want2: &ds.CustomerMetadata{
				Customer: getCustomerRef(),
				ServiceAccounts: []ds.ServiceAccountMetadata{
					createServiceAccount(2, "0000-1111-2222-3333", projectID),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := acquireServiceAccount(tt.args.reserved, tt.args.acquired, tt.args.serviceAccountEmail, tt.args.billingAccountID); err != nil {
				if !tt.wantErr {
					t.Errorf("acquireServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if len(tt.args.reserved.ServiceAccounts) != len(tt.want1.ServiceAccounts) {
				t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", tt.args.reserved.ServiceAccounts, tt.want1.ServiceAccounts)
			}

			for i, sa := range tt.args.reserved.ServiceAccounts {
				if sa.ServiceAccountEmail != tt.want1.ServiceAccounts[i].ServiceAccountEmail {
					t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", tt.args.reserved.ServiceAccounts, tt.want1.ServiceAccounts)
					break
				}
			}

			if len(tt.args.acquired.ServiceAccounts) != len(tt.want2.ServiceAccounts) {
				t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", tt.args.reserved.ServiceAccounts, tt.want2.ServiceAccounts)
			}

			for i, sa := range tt.args.acquired.ServiceAccounts {
				if sa.ServiceAccountEmail != tt.want2.ServiceAccounts[i].ServiceAccountEmail {
					t.Errorf("getReservedServiceAccountEmail() got = %v, want %v", tt.args.acquired.ServiceAccounts, tt.want2.ServiceAccounts)
					break
				}
			}
			// if tt.args.acquired.Customer == nil {
			// 	t.Errorf("getReservedServiceAccountEmail() got1 = %v, want %v", "nil pointer", tt.args.customerRef.ID)
			// }
			// if tt.args.acquired.Customer != nil && tt.args.acquired.Customer.ID != tt.want3.Customer.ID {
			// 	t.Errorf("getReservedServiceAccountEmail() got1 = %v, want %v", tt.args.customerRef, tt.want3.Customer)
			// }
		})
	}
}

func Test_projectIDFromServiceAccountEmail(t *testing.T) {
	type args struct {
		serviceAccountEmail string
	}

	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "correct project id",
			args: args{
				serviceAccountEmail: getServiceAccountEmail(1),
			},
			want: projectID,
		},
		{
			name: "invalid service account email",
			args: args{
				serviceAccountEmail: "invalid.email.com",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := projectIDFromServiceAccountEmail(tt.args.serviceAccountEmail)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("projectIDFromServiceAccountEmail() error = %v, wantErr %v", err, tt.wantErr)
				}

				return
			}

			if got != tt.want {
				t.Errorf("projectIDFromServiceAccountEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}
