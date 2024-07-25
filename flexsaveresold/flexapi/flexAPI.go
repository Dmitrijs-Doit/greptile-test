package flexapi

import (
	"context"
	"errors"
	"fmt"
	stdhttp "net/http"
	"sort"
	"sync"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/shared"
	"github.com/doitintl/http"
)

var (
	ErrRecommendationsNotFound = errors.New("recommendations not found")
)

//go:generate mockery --name FlexAPI --output ./mocks
type FlexAPI interface {
	ListFlexsaveAccounts(ctx context.Context) ([]string, error)
	ListFlexsaveAccountsWithCache(ctx context.Context, refreshTime time.Duration) ([]string, error)
	ListFullFlexsaveAccounts(ctx context.Context) ([]*Account, error)
	ListARNs(ctx context.Context) ([]string, error)
	GetRDSPayerRecommendations(ctx context.Context, payerID string) ([]RDSBottomUpRecommendation, error)
}

type Service struct {
	flexAPIClient       http.IClient
	timeUpdatedAccounts time.Time
	flexsaveAccounts    []*Account
	mu                  *sync.Mutex
}

func NewFlexAPIService() (FlexAPI, error) {
	ctx := context.Background()
	baseURL := shared.GetFlexAPIURL()

	tokenSource, err := shared.GetTokenSource(ctx)
	if err != nil {
		return nil, err
	}

	client, err := http.NewClient(ctx, &http.Config{
		BaseURL:     baseURL,
		TokenSource: tokenSource,
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		client,
		time.Time{},
		nil,
		&sync.Mutex{},
	}, nil
}

func (s *Service) GetRDSPayerRecommendations(ctx context.Context, payerID string) ([]RDSBottomUpRecommendation, error) {
	var recommendations []RDSBottomUpRecommendation

	if _, err := s.flexAPIClient.Get(ctx, &http.Request{
		URL:          fmt.Sprintf("v2/payer/%s/rds-recommendation", payerID),
		ResponseType: &recommendations,
	}); err != nil {
		webErr, ok := err.(http.WebError)
		if ok && webErr.ErrorCode() == stdhttp.StatusNotFound {
			return nil, ErrRecommendationsNotFound
		}

		return nil, err
	}

	return recommendations, nil
}

func (s *Service) getAccountIDs(accounts []*Account) []string {
	result := make([]string, 0, len(accounts))

	for _, a := range accounts {
		result = append(result, a.AccountID)
	}

	return result
}

// ListFullFlexsaveAccounts returns the most recent list of flexsave accounts
func (s *Service) ListFullFlexsaveAccounts(ctx context.Context) ([]*Account, error) {
	var accounts []*Account

	if _, err := s.flexAPIClient.Get(ctx, &http.Request{
		URL:          "/accounts",
		ResponseType: &accounts,
	}); err != nil {
		return nil, err
	}

	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].AccountID < accounts[j].AccountID
	})

	s.updateAccountsCache(accounts)

	return accounts, nil
}

// ListFlexsaveAccounts returns the most recent list of flexsave account ids
func (s *Service) ListFlexsaveAccounts(ctx context.Context) ([]string, error) {
	accounts, err := s.ListFullFlexsaveAccounts(ctx)
	if err != nil {
		return nil, err
	}

	return s.getAccountIDs(accounts), nil
}

// ListFlexsaveAccountsWithCache returns a cached list of flexsave account ids if the cache is still fresh
// according to the refreshTime parameter. Otherwise, it will fetch the latest list of flexsave account ids
func (s *Service) ListFlexsaveAccountsWithCache(ctx context.Context, refreshTime time.Duration) ([]string, error) {
	if time.Since(s.timeUpdatedAccounts) < refreshTime {
		return s.getAccountIDs(s.flexsaveAccounts), nil
	}

	return s.ListFlexsaveAccounts(ctx)
}

func (s *Service) ListARNs(ctx context.Context) ([]string, error) {
	var arns []string

	if _, err := s.flexAPIClient.Get(ctx, &http.Request{
		URL:          "/accounts/resource-names",
		ResponseType: &arns,
	}); err != nil {
		return nil, err
	}

	return arns, nil
}

func (s *Service) updateAccountsCache(accounts []*Account) {
	s.mu.Lock()
	s.timeUpdatedAccounts = time.Now()
	s.flexsaveAccounts = make([]*Account, len(accounts))
	copy(s.flexsaveAccounts, accounts)
	s.mu.Unlock()
}
