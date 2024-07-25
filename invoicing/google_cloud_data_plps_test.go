package invoicing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDalMocks "github.com/doitintl/hello/scheduled-tasks/contract/dal/mocks"
	contractDomain "github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func Test_InvoiceService_calculateNewPLPSCost(t *testing.T) {
	type fields struct {
	}

	type args struct {
		queryProjectRow      *QueryProjectRow
		gcpPLPSChargePercent float64
		plpsCharges          domain.SortablePLPSCharges
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    float64
		wantErr error
	}{
		{
			name: "get a new percent from a second contract",
			args: args{
				queryProjectRow: &QueryProjectRow{
					Date: civil.Date{
						Year:  2023,
						Month: 8,
						Day:   4,
					},
					Cost: 30,
				},
				gcpPLPSChargePercent: 3,
				plpsCharges: domain.SortablePLPSCharges{
					{
						PLPSPercent: 5,
						StartDate:   time.Date(2022, time.September, 10, 23, 0, 0, 0, time.UTC),
						EndDate:     time.Date(2022, time.November, 31, 23, 0, 0, 0, time.UTC),
					},
					{
						PLPSPercent: 6,
						StartDate:   time.Date(2022, time.December, 1, 0, 0, 0, 0, time.UTC),
						EndDate:     time.Date(2023, time.September, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			want:    float64(60),
			wantErr: nil,
		},
		{
			name: "get a new percent from a first contract",
			args: args{
				queryProjectRow: &QueryProjectRow{
					Date: civil.Date{
						Year:  2022,
						Month: 11,
						Day:   4,
					},
					Cost: 30,
				},
				gcpPLPSChargePercent: 3,
				plpsCharges: domain.SortablePLPSCharges{
					{
						PLPSPercent: 5,
						StartDate:   time.Date(2022, time.September, 10, 23, 0, 0, 0, time.UTC),
						EndDate:     time.Date(2022, time.November, 31, 23, 0, 0, 0, time.UTC),
					},
					{
						PLPSPercent: 6,
						StartDate:   time.Date(2022, time.December, 1, 0, 0, 0, 0, time.UTC),
						EndDate:     time.Date(2023, time.September, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			want:    float64(50),
			wantErr: nil,
		},
		{
			name: "error when suitable contract is not found",
			args: args{
				queryProjectRow: &QueryProjectRow{
					Date: civil.Date{
						Year:  2025,
						Month: 11,
						Day:   4,
					},
					Cost: 30,
				},
				gcpPLPSChargePercent: 3,
				plpsCharges: domain.SortablePLPSCharges{
					{
						PLPSPercent: 5,
						StartDate:   time.Date(2022, time.September, 10, 23, 0, 0, 0, time.UTC),
						EndDate:     time.Date(2022, time.November, 31, 23, 0, 0, 0, time.UTC),
					},
					{
						PLPSPercent: 6,
						StartDate:   time.Date(2022, time.December, 1, 0, 0, 0, 0, time.UTC),
						EndDate:     time.Date(2023, time.September, 1, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			want:    float64(30),
			wantErr: ErrNoSuitablePLPSContractFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &InvoicingService{}
			got, gotErr := s.calculateNewPLPSCost(
				tt.args.queryProjectRow,
				tt.args.gcpPLPSChargePercent,
				tt.args.plpsCharges,
			)

			if tt.wantErr != nil {
				assert.ErrorContains(t, gotErr, tt.wantErr.Error())
			} else {
				assert.NoError(t, gotErr)
			}

			assert.Equal(t, tt.want, got, "calculateNewPLPSCost")
		})
	}
}

func Test_InvoiceService_getPLPSCharges(t *testing.T) {
	type fields struct {
		contractDAL *contractDalMocks.ContractFirestore
	}

	type args struct {
		customerRef        *firestore.DocumentRef
		assetRef           *firestore.DocumentRef
		invoicingStartDate time.Time
		invoicingEndDate   time.Time
	}

	customerID := "some customer id"
	customerRef := &firestore.DocumentRef{
		ID: customerID,
	}

	assetID := "some asset ref id"
	assetRef := &firestore.DocumentRef{
		ID: assetID,
	}

	someOtherAssetRef := &firestore.DocumentRef{
		ID: "some other ref",
	}

	contractBeforeInvoiceRange := common.Contract{
		Customer: customerRef,
		Active:   true,
		Assets: []*firestore.DocumentRef{
			assetRef,
			someOtherAssetRef,
		},
		StartDate:   time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2022, time.July, 31, 0, 0, 0, 0, time.UTC),
		PLPSPercent: 3,
	}

	contractWithinInvoiceRangeWithDifferentAsset := common.Contract{
		Customer: customerRef,
		Active:   true,
		Assets: []*firestore.DocumentRef{
			someOtherAssetRef,
		},
		StartDate:   time.Date(2022, time.August, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2022, time.September, 15, 0, 0, 0, 0, time.UTC),
		PLPSPercent: 4,
	}

	contractWithinInvoiceRange1 := common.Contract{
		Customer: customerRef,
		Active:   true,
		Assets: []*firestore.DocumentRef{
			assetRef,
			someOtherAssetRef,
		},
		StartDate:   time.Date(2022, time.August, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2022, time.September, 15, 0, 0, 0, 0, time.UTC),
		PLPSPercent: 5,
	}

	contractWithinInvoiceRange2 := common.Contract{
		Customer: customerRef,
		Active:   true,
		Assets: []*firestore.DocumentRef{
			assetRef,
		},
		StartDate:   time.Date(2022, time.September, 15, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2022, time.September, 25, 0, 0, 0, 0, time.UTC),
		PLPSPercent: 6,
	}

	contractWithinInvoiceRangeOnDemand := common.Contract{
		Customer: customerRef,
		Active:   true,
		Assets: []*firestore.DocumentRef{
			assetRef,
		},
		StartDate:   time.Date(2022, time.September, 25, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Time{},
		PLPSPercent: 7,
	}

	contractWithinInvoiceRangeInactive := common.Contract{
		Customer: customerRef,
		Active:   false,
		Assets: []*firestore.DocumentRef{
			assetRef,
		},
		StartDate:   time.Date(2022, time.September, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2023, time.September, 1, 0, 0, 0, 0, time.UTC),
		PLPSPercent: 9,
	}

	plpsCharge1 := domain.PLPSCharge{
		PLPSPercent: 5,
		StartDate:   time.Date(2022, time.August, 1, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2022, time.September, 15, 0, 0, 0, 0, time.UTC),
	}

	plpsCharge2 := domain.PLPSCharge{
		PLPSPercent: 6,
		StartDate:   time.Date(2022, time.September, 15, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2022, time.September, 25, 0, 0, 0, 0, time.UTC),
	}

	plpsCharge3 := domain.PLPSCharge{
		PLPSPercent: 7,
		StartDate:   time.Date(2022, time.September, 25, 0, 0, 0, 0, time.UTC),
		EndDate:     time.Date(2022, time.October, 1, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    domain.SortablePLPSCharges
		wantErr error
		on      func(*fields)
	}{
		{
			name: "get sorted plps charges",
			args: args{
				customerRef:        customerRef,
				assetRef:           assetRef,
				invoicingStartDate: time.Date(2022, time.September, 1, 0, 0, 0, 0, time.UTC),
				invoicingEndDate:   time.Date(2022, time.September, 30, 0, 0, 0, 0, time.UTC),
			},
			on: func(f *fields) {
				f.contractDAL.On(
					"GetContractsByType",
					testutils.ContextBackgroundMock,
					customerRef,
					contractDomain.ContractTypeGoogleCloudPLPS,
				).
					Return([]common.Contract{
						contractBeforeInvoiceRange,
						contractWithinInvoiceRangeWithDifferentAsset,
						contractWithinInvoiceRange1,
						contractWithinInvoiceRange2,
						contractWithinInvoiceRangeInactive,
						contractWithinInvoiceRangeOnDemand,
					}, nil).
					Once()
			},
			want: domain.SortablePLPSCharges{
				&plpsCharge1,
				&plpsCharge2,
				&plpsCharge3,
			},
			wantErr: nil,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				contractDAL: contractDalMocks.NewContractFirestore(t),
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			s := &InvoicingService{
				contractDAL: tt.fields.contractDAL,
			}

			got, gotErr := s.getPLPSCharges(
				ctx,
				tt.args.customerRef,
				tt.args.assetRef,
				tt.args.invoicingStartDate,
				tt.args.invoicingEndDate,
			)

			if tt.wantErr != nil {
				assert.ErrorContains(t, gotErr, tt.wantErr.Error())
			} else {
				assert.NoError(t, gotErr)
			}

			for _, g := range got {
				fmt.Printf("%+v\n", g)
			}

			assert.Equal(t, len(tt.want), len(got), "len is wrong")

			for i := range got {
				assert.Equal(t, tt.want[i], got[i], "getPLPSCharge")
			}
		})
	}
}
