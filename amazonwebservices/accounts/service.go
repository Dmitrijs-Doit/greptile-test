package accounts

import (
	"context"
	"errors"
	"math"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate mockery --name Service --inpackage
type Service interface {
	GetOldestJoinTimestampAge(ctx context.Context, ids []string, now time.Time) (int, error)
}

type service struct {
	dal Dal
}

var errNotFound = errors.New("no valid accounts found for given asset IDs")

func NewService() (Service, error) {
	fs, err := firestore.NewClient(context.Background(), common.ProjectID)
	if err != nil {
		return nil, err
	}

	return &service{
		newDal(fs),
	}, nil
}

func (s service) GetOldestJoinTimestampAge(ctx context.Context, ids []string, now time.Time) (int, error) {
	var maxDays float64 = -1

	for _, id := range ids {
		account, err := s.dal.findAccountByID(ctx, id)
		if err != nil && status.Code(err) != codes.NotFound {
			return -1, err
		}

		if err != nil {
			continue
		}

		diff := now.Sub(account.JoinedTimestamp)
		ageInDays := diff.Hours() / 24

		if ageInDays > maxDays {
			maxDays = ageInDays
		}
	}

	if maxDays > -1 {
		return int(math.Round(maxDays)), nil
	}

	return int(maxDays), errNotFound
}
