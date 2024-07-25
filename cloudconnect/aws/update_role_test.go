package aws

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	dal "github.com/doitintl/hello/scheduled-tasks/cloudconnect/dal"
	dalMocks "github.com/doitintl/hello/scheduled-tasks/cloudconnect/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestPermissions_SendWelcomeEmail(t *testing.T) {
	ctx := context.Background()
	req := RoleRequest{
		CustomerID: "EE8CtpzYiKp0dVAESVrB",
		Arn:        "arn:aws:iam::126166551120:role/doitintl-cmp",
		AccountID:  "126166551120",
		StackID:    "AROAR2YA4ZJIFPU24GQ72",
		StackName:  "DoitCmpNewRole-1661349691123",
	}

	type fields struct {
		dal *dalMocks.IAwsConnect
	}

	type args struct {
		ctx context.Context
		req *RoleRequest
	}

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "send email to all admins with AMs in BCC and updage flags",
			fields: fields{
				dal: dalMocks.NewIAwsConnect(t),
			},
			args: args{
				ctx: ctx,
				req: &req,
			},
			on: func(f *fields) {
				f.dal.On("GetSpot0CustomerFlags", ctx, req.CustomerID).Return(nil, nil)
				f.dal.On("GetCustomer", ctx, req.CustomerID).Return(&common.Customer{
					Name: "Test Customer",
				}, nil)
				f.dal.On("GetCustomerAccountManagers", ctx, mock.Anything, mock.Anything).Return([]*common.AccountManager{
					{
						Email: "am1@doit.com",
					},
					{
						Email: "am2@doit.com",
					},
				}, nil)
				f.dal.On("GetCustomerAdmins", ctx, req.CustomerID).Return([]common.User{
					{
						Email:     "admin1@doit.com",
						FirstName: "Admin1",
					},
					{
						Email:     "admin2@doit.com",
						FirstName: "Admin2",
					},
				}, nil)
				f.dal.On(
					"SendMail",
					ctx,
					[]dal.MailRecipient{
						{
							Email:     "admin1@doit.com",
							FirstName: "Admin1",
						},
						{
							Email:     "admin2@doit.com",
							FirstName: "Admin2",
						},
					},
					[]string{"am1@doit.com", "am2@doit.com"},
					"Test Customer",
				).Return(nil)
				f.dal.On(
					"SetSpot0CustomerFlags",
					ctx,
					req.CustomerID,
				).Return(nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.on != nil {
				tt.on(&tt.fields)
			}

			p := &Permissions{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &loggerMocks.ILogger{}
				},
				fs:      nil,
				bq:      nil,
				session: nil,
				dal:     tt.fields.dal,
			}
			err := p.SendWelcomeEmail(tt.args.ctx, tt.args.req)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
