package handlers

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/domain"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/service"
	mocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/service/mocks"
	invoicingMocks "github.com/doitintl/hello/scheduled-tasks/invoicing/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/stretchr/testify/mock"
)

func Test_parsePLESFile(t *testing.T) {
	type args struct {
		file *os.File
	}

	expectedTime, err := time.Parse("2006-01", "2024-01")
	if err != nil {
		t.Errorf("error parsing time: %s", err)
	}

	tests := []struct {
		name  string
		args  args
		want  []domain.PLESAccount
		want1 []error
	}{
		{
			name: "valid file",
			args: args{
				file: createTestFile([][]string{
					{"account_name", "account_id", "support_level", "payer_id"},
					{"test1", "012345678910", "basic", "109876543210"},
					{"test2", "123456789012", "business", "210987654321"},
				}),
			},
			want: []domain.PLESAccount{
				{
					AccountName:  "test1",
					AccountID:    "012345678910",
					SupportLevel: "basic",
					PayerID:      "109876543210",
					InvoiceMonth: expectedTime,
				},
				{
					AccountName:  "test2",
					AccountID:    "123456789012",
					SupportLevel: "business",
					PayerID:      "210987654321",
					InvoiceMonth: expectedTime,
				},
			},
			want1: []error{},
		},
		{
			name: "invalid file",
			args: args{
				file: createTestFile([][]string{
					{"account_name", "account_id", "support_level", "payer_id", ""},
					{"test1", "012345678910", "basic", "109876543210"},
					{"test2", "123456789012", "business"},
				}),
			},
			want: nil,
			want1: []error{
				fmt.Errorf("invalid CSV file: expected 4 columns, got 5"),
			},
		},
		{
			name: "empty file",
			args: args{
				file: createTestFile([][]string{}),
			},
			want: nil,
			want1: []error{
				fmt.Errorf("error reading CSV: EOF"),
			},
		},
	}
	for _, tt := range tests {
		defer os.Remove(tt.args.file.Name())

		t.Run(tt.name, func(t *testing.T) {
			timeNow := time.Now()
			got, got1 := parsePLESFile(tt.args.file, "2024-01")

			for i, account := range got {
				if !reflect.DeepEqual(account.AccountID, tt.want[i].AccountID) ||
					!reflect.DeepEqual(account.AccountName, tt.want[i].AccountName) ||
					!reflect.DeepEqual(account.InvoiceMonth, tt.want[i].InvoiceMonth) ||
					!reflect.DeepEqual(account.PayerID, tt.want[i].PayerID) ||
					!reflect.DeepEqual(account.SupportLevel, tt.want[i].SupportLevel) ||
					!timeNow.Before(account.UpdateTime) {
					t.Errorf("parsePLESFile() got = %v, want %v", got, tt.want)
				}
			}

			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("parsePLESFile() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func createTestFile(content [][]string) *os.File {
	file, _ := os.CreateTemp("./", "testfile")
	writer := csv.NewWriter(file)

	for _, row := range content {
		writer.Write(row)
	}

	writer.Flush()
	file.Seek(0, 0)

	return file
}

func Test_validateHeaders(t *testing.T) {
	type args struct {
		csvReader *csv.Reader
	}

	tests := []struct {
		name string
		args args
		want []error
	}{
		{
			name: "valid headers",
			args: args{
				csvReader: csv.NewReader(strings.NewReader("account_name,account_id,support_level,payer_id\n")),
			},
			want: nil,
		},
		{
			name: "missing headers",
			args: args{
				csvReader: csv.NewReader(strings.NewReader("header1,header3\n")),
			},
			want: []error{fmt.Errorf("invalid CSV file: expected 4 columns, got 2")},
		},
		{
			name: "extra headers",
			args: args{
				csvReader: csv.NewReader(strings.NewReader("header1,header2,header3,header4,header5\n")),
			},
			want: []error{fmt.Errorf("invalid CSV file: expected 4 columns, got 5")},
		},
		{
			name: "incorrect order of headers",
			args: args{
				csvReader: csv.NewReader(strings.NewReader("account_name,account_id,payer_id,support_level\n")),
			},
			want: []error{ErrInvalidHeaders},
		},
		{
			name: "empty file",
			args: args{
				csvReader: csv.NewReader(strings.NewReader("\n")),
			},
			want: []error{fmt.Errorf("error reading CSV: EOF")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateHeaders(tt.args.csvReader); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("validateHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateRow(t *testing.T) {
	type args struct {
		record   []string
		rowIndex int
	}

	tests := []struct {
		name string
		args args
		want []error
	}{
		{
			name: "valid row",
			args: args{
				record:   []string{"account1", "012345678910", "basic", "109876543210"},
				rowIndex: 1,
			},
			want: []error{},
		},
		{
			name: "missing columns",
			args: args{
				record:   []string{"account1", "012345678910"},
				rowIndex: 2,
			},
			want: []error{ErrInvalidNumberOfColumns(2)},
		},
		{
			name: "extra columns",
			args: args{
				record:   []string{"account1", "012345678910", "basic", "109876543210", "extra"},
				rowIndex: 3,
			},
			want: []error{ErrInvalidNumberOfColumns(3)},
		},
		{
			name: "empty row",
			args: args{
				record:   []string{},
				rowIndex: 4,
			},
			want: []error{ErrInvalidNumberOfColumns(4)},
		},
		{
			name: "all invalid fields",
			args: args{
				record:   []string{"", "", "", ""},
				rowIndex: 5,
			},
			want: []error{
				ErrInvalidAccountName(5),
				ErrInvalidAccountIDFormat(5, ""),
				ErrInvalidSupportLevel(5, ""),
				ErrInvalidPayerIDFormat(5, ""),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validateRow(tt.args.record, tt.args.rowIndex)
			if len(got) != len(tt.want) {
				t.Errorf("validateRow() = %v, want %v", got, tt.want)
			} else {
				for i := range got {
					if got[i].Error() != tt.want[i].Error() {
						t.Errorf("validateRow() = %v, want %v", got, tt.want)
						break
					}
				}
			}
		})
	}
}

func Test_validatePayerID(t *testing.T) {
	type args struct {
		payerID string
		row     int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid payer ID",
			args: args{
				payerID: "012345678910",
				row:     0,
			},
			wantErr: false,
		},
		{
			name: "invalid payer ID",
			args: args{
				payerID: "123",
				row:     1,
			},
			wantErr: true,
		},
		{
			name: "empty payer ID",
			args: args{
				payerID: "",
				row:     2,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePayerID(tt.args.payerID, tt.args.row); (err != nil) != tt.wantErr {
				t.Errorf("validatePayerID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateAccountName(t *testing.T) {
	type args struct {
		accountName string
		row         int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "valid account name",
			args: args{
				accountName: "test",
				row:         0,
			},
			wantErr: false,
		},
		{
			name: "empty account name",
			args: args{
				accountName: "",
				row:         1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateAccountName(tt.args.accountName, tt.args.row); (err != nil) != tt.wantErr {
				t.Errorf("validateAccountName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateAccountID(t *testing.T) {
	type args struct {
		accountID string
		row       int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid account ID",
			args: args{
				accountID: "123",
				row:       0,
			},
			wantErr: true,
		},
		{
			name: "empty account ID",
			args: args{
				accountID: "",
				row:       1,
			},
			wantErr: true,
		},
		{
			name: "valid account ID",
			args: args{
				accountID: "012345678910",
				row:       2,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateAccountID(tt.args.accountID, tt.args.row); (err != nil) != tt.wantErr {
				t.Errorf("validateAccountID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateSupportLevel(t *testing.T) {
	type args struct {
		supportLevel string
		row          int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "invalid support level",
			args: args{
				supportLevel: "invalid",
				row:          0,
			},
			wantErr: true,
		},
		{
			name: "empty support level",
			args: args{
				supportLevel: "",
				row:          1,
			},
			wantErr: true,
		},
		{
			name: "valid support level: basic",
			args: args{
				supportLevel: "basic",
				row:          2,
			},
			wantErr: false,
		},
		{
			name: "valid support level: business",
			args: args{
				supportLevel: "business",
				row:          3,
			},
			wantErr: false,
		},
		{
			name: "valid support level: developer",
			args: args{
				supportLevel: "developer",
				row:          4,
			},
			wantErr: false,
		},
		{
			name: "valid support level: enterprise",
			args: args{
				supportLevel: "enterprise",
				row:          5,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateSupportLevel(tt.args.supportLevel, tt.args.row); (err != nil) != tt.wantErr {
				t.Errorf("validateSupportLevel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPLES_validateInvoiceMonth(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        service.PLESIface
		billingData    *invoicingMocks.BillingData
	}

	type args struct {
		ctx   context.Context
		month string
	}

	currentTime := time.Now()
	currentMonthStr := currentTime.Format("2006-01")
	firstDayCurrentMonth, _ := time.Parse("2006-01", currentMonthStr)
	previousMonthDate := firstDayCurrentMonth.AddDate(0, -1, 0)
	previousMonthStr := previousMonthDate.Format("2006-01")

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name: "invalid month format",
			args: args{
				ctx:   context.TODO(),
				month: "2022-13",
			},
			wantErr: true,
		},
		{
			name: "valid month format",
			args: args{
				ctx:   context.TODO(),
				month: currentMonthStr,
			},
			wantErr: false,
		},
		{
			name: "empty month",
			args: args{
				ctx:   context.TODO(),
				month: "",
			},
			wantErr: true,
		},
		{
			name: "previous month",
			args: args{
				ctx:   context.TODO(),
				month: previousMonthStr,
			},
			on: func(f *fields) {
				f.billingData.On("HasAnyInvoiceBeenIssued", mock.Anything, mock.Anything).Return(false, nil)
			},
			wantErr: false,
		},
		{
			name: "previous month invoice already issued",
			args: args{
				ctx:   context.TODO(),
				month: previousMonthStr,
			},
			on: func(f *fields) {

				f.billingData.On("HasAnyInvoiceBeenIssued", mock.Anything, mock.Anything).Return(true, nil)
			},
			wantErr: true,
		},
		{
			name: "previous month HasAnyInvoiceBeenIssued returns error",
			args: args{
				ctx:   context.TODO(),
				month: previousMonthStr,
			},
			on: func(f *fields) {

				f.billingData.On("HasAnyInvoiceBeenIssued", mock.Anything, mock.Anything).Return(true, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				logger.FromContext,
				&mocks.PLESIface{},
				&invoicingMocks.BillingData{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			h := &PLES{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
				billingData:    tt.fields.billingData,
			}
			if err := h.validateInvoiceMonth(tt.args.ctx, tt.args.month); (err != nil) != tt.wantErr {
				t.Errorf("PLES.validateInvoiceMonth() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
