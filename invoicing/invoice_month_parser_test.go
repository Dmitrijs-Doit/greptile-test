package invoicing

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

func Test_defaultInvoiceMonthParser_GetInvoiceMonth(t *testing.T) {
	now := time.Now().UTC()
	testInvoiceMonth := "2022-01-01"
	futureMonth := "2025-01-01"
	invalidMonth := "25a"
	parsedMonth, _ := time.Parse("2006-01-02", testInvoiceMonth)

	type fields struct {
		InvoicingDaySwitchOver int
	}

	type args struct {
		invoiceMonthInput string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    time.Time
		wantErr error
	}{
		{
			name:    "",
			args:    args{invoiceMonthInput: "2022-01-01"},
			want:    time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC),
			wantErr: nil,
		},
		{
			name: "valid month",
			args: args{
				invoiceMonthInput: testInvoiceMonth,
			},
			want:    parsedMonth,
			wantErr: nil,
		},
		{
			name: "invalid month string",
			args: args{
				invoiceMonthInput: invalidMonth,
			},
			want:    time.Time{},
			wantErr: errors.New(`parsing time "25a" as "2006-01-02": cannot parse "25a" as "2006"`),
		},
		{
			name: "future invalid month",
			args: args{
				invoiceMonthInput: futureMonth,
			},
			want:    time.Time{},
			wantErr: web.ErrBadRequest,
		},
		{
			name:   "no invoice month before switch over",
			fields: fields{InvoicingDaySwitchOver: 32},
			args: args{
				invoiceMonthInput: "",
			},
			want:    time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC),
			wantErr: nil,
		},
		{
			name:   "no invoice month after switch over",
			fields: fields{InvoicingDaySwitchOver: 0},
			args: args{
				invoiceMonthInput: "",
			},
			want:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC),
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &DefaultInvoiceMonthParser{
				InvoicingDaySwitchOver: tt.fields.InvoicingDaySwitchOver,
			}

			got, err := s.GetInvoiceMonth(tt.args.invoiceMonthInput)
			if tt.wantErr != nil {
				assert.Errorf(t, err, tt.wantErr.Error(), fmt.Sprintf("GetInvoiceMonth(%v)", tt.args.invoiceMonthInput))
			}

			assert.Equalf(t, tt.want, got, "GetInvoiceMonth(%v)", tt.args.invoiceMonthInput)
		})
	}
}
