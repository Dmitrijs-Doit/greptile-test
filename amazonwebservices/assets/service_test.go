package assets

import (
	"context"
	"testing"

	"github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestNewAWSAssetsService(t *testing.T) {
	type args struct {
		log             logger.Provider
		conn            *connection.Connection
		cloudTaskClient iface.CloudTaskClient
	}

	ctx := context.Background()

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		t.Errorf("main: could not initialize logging. error %s", err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		t.Errorf("main: could not initialize db connections. error %s", err)
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestNewAWSAssetsService",
			args: args{
				log:             nil,
				conn:            conn,
				cloudTaskClient: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAWSAssetsService(tt.args.log, tt.args.conn, tt.args.cloudTaskClient)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAWSAssetsService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
