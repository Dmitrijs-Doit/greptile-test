package announcekit

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/http"
	mockClient "github.com/doitintl/http/mocks"
	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	dummySecretKey = "thisisadummysecretkey1234thisisadummysecret"
	alg            = "HS256"
)

func createTestService(ctx context.Context, t *testing.T, secretKey string) *AnnounceKitService {
	return &AnnounceKitService{
		logger.FromContext,
		[]byte(secretKey),
		&mockClient.IClient{},
	}
}

func TestCreateAnnouncekitToken(t *testing.T) {
	ctx := context.Background()
	testService := createTestService(ctx, t, dummySecretKey)
	userClaims := JwtUserClaims{
		ID:    "12345dummyUserId",
		EMAIL: "dummyEmail@domain.com",
		NAME:  "fName lName",
	}

	tokenString, err := testService.CreateAuthToken(ctx, &userClaims)
	if err != nil {
		t.Fatalf("failed to create token with: %v\n", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(dummySecretKey), nil
	})
	if !token.Valid {
		t.Fatal("invalid token")
	}

	if token.Method.Alg() != alg {
		t.Fatalf("expected alg is %s but got %s\n", alg, token.Method.Alg())
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if claims["id"] != userClaims.ID || claims["email"] != userClaims.EMAIL || claims["name"] != userClaims.NAME {
			t.Fatal("parsed claims differ")
		}
	} else {
		t.Fatal("invalid claims")
	}
}

func TestAnnounceKitService_GetChangeLogs(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		jwtKeyB        []byte
		client         *mockClient.IClient
	}

	feed := AnnoucekitFeed{
		Items: []ChangeLogItem{
			{
				Title:   "First item title",
				Summary: "First item summary",
				URL:     "First item URL", DateModified: time.Date(2022, time.August, 4, 12, 24, 36, 250000000, time.UTC),
			},
			{
				Title:        "Second item title",
				Summary:      "Second item summary",
				URL:          "Second item URL",
				DateModified: time.Date(2022, time.July, 4, 12, 24, 36, 250000000, time.UTC),
			},
		},
	}

	getFailError := errors.New("failed to get")

	type args struct {
		ctx       context.Context
		startDate time.Time
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		want    AnnoucekitFeed
	}{
		{
			name: "returns logs with DateModified after startDate",
			args: args{
				startDate: time.Date(2022, time.August, 1, 0, 0, 0, 0, time.UTC),
				ctx:       context.Background(),
			},
			on: func(f *fields) {
				f.client.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					*request.ResponseType.(*AnnoucekitFeed) = feed
				}).Once()
			},
			want: AnnoucekitFeed{
				Items: []ChangeLogItem{
					{
						Title:         "First item title",
						Summary:       "First item summary",
						URL:           "First item URL",
						DateModified:  time.Date(2022, time.August, 4, 12, 24, 36, 250000000, time.UTC),
						DateFormatted: "August 4, 2022",
					},
				},
			},
		},
		{
			name: "returns error when failed to get",
			args: args{
				startDate: time.Date(2022, time.August, 1, 0, 0, 0, 0, time.UTC),
				ctx:       context.Background(),
			},
			on: func(f *fields) {
				f.client.On("Get", mock.Anything, mock.Anything).Return(nil, getFailError).Once()
			},
			wantErr: getFailError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				client: &mockClient.IClient{},
			}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &AnnounceKitService{
				loggerProvider: fields.loggerProvider,
				jwtKeyB:        fields.jwtKeyB,
				client:         fields.client,
			}

			got, err := s.GetChangeLogs(tt.args.ctx, tt.args.startDate)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AnnounceKitService.GetChangeLogs() = %v, want %v", got, tt.want)
			}
		})
	}
}
