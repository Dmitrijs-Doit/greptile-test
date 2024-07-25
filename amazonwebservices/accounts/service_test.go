package accounts

import (
	"context"
	"errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_service_GetOldestJoinTimestampAge(t *testing.T) {
	mockDal := &MockDal{}
	ctx := context.Background()
	err := errors.New("oh no")
	now := time.Date(
		2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

	type args struct {
		ctx context.Context
		ids []string
	}

	tests := []struct {
		name string
		args args
		want int
		on   func(m *MockDal)
		err  error
	}{
		{
			name: "service returns error if dal does also",
			args: args{
				ctx: ctx,
				ids: []string{"id1"},
			},
			want: -1,
			err:  err,
			on: func(m *MockDal) {
				m.On("findAccountByID", ctx, "id1").
					Return(nil, err).
					Once()
			},
		},
		{
			name: "service returns error if no accounts are found",
			args: args{
				ctx: ctx,
				ids: []string{"id1"},
			},
			want: -1,
			err:  errNotFound,
			on: func(m *MockDal) {
				m.On("findAccountByID", ctx, "id1").
					Return(nil, status.Error(codes.NotFound, "account not found")).
					Once()
			},
		},
		{
			name: "service returns diffed time",
			args: args{
				ctx: ctx,
				ids: []string{"id1"},
			},
			want: 15,
			err:  nil,
			on: func(m *MockDal) {
				m.On("findAccountByID", ctx, "id1").
					Return(&Account{JoinedTimestamp: now.AddDate(0, 0, -15)}, nil).
					Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := service{
				dal: mockDal,
			}

			if tt.on != nil {
				tt.on(mockDal)
			}

			got, err := s.GetOldestJoinTimestampAge(tt.args.ctx, tt.args.ids, now)
			assert.Equal(t, tt.err, err)
			assert.Equalf(t, tt.want, got, "GetOldestJoinTimestampAge(%v, %v)", tt.args.ctx, tt.args.ids)
		})
	}
}
