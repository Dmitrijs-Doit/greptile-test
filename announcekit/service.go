package announcekit

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/golang-jwt/jwt"

	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/http"
)

const changeLogBaseURL = "https://changelog.doit.com/"

type AnnounceKit interface {
	GetChangeLogs(ctx context.Context, startDate time.Time) (AnnoucekitFeed, error)
	CreateAuthToken(ctx context.Context, userClaims *JwtUserClaims) (string, error)
}

type ChangeLogItem struct {
	Title         string    `json:"title"`
	Summary       string    `json:"summary"`
	URL           string    `json:"url"`
	DateModified  time.Time `json:"date_modified"`
	DateFormatted string    `json:"date_formatted"`
}

type AnnoucekitFeed struct {
	Items []ChangeLogItem `json:"items"`
}

type JwtUserClaims struct {
	ID    string
	EMAIL string
	NAME  string
}

type secret struct {
	JwtKey string `json:"jwt_key"`
}

type AnnounceKitService struct {
	loggerProvider logger.Provider
	jwtKeyB        []byte
	client         http.IClient
}

func NewAnnounceKitService(loggerProvider logger.Provider) (*AnnounceKitService, error) {
	secret := getSecret()
	jwtKeyB := []byte(secret.JwtKey)
	client, err := http.NewClient(context.Background(), &http.Config{BaseURL: changeLogBaseURL})

	if err != nil {
		return nil, err
	}

	return &AnnounceKitService{
		loggerProvider,
		jwtKeyB,
		client,
	}, nil
}

func getSecret() *secret {
	var secret secret

	secretB, err := secretmanager.AccessSecretLatestVersion(context.Background(), secretmanager.SecretAnnouncekit)
	if err != nil || secretB == nil {
		panic(err)
	}

	if err := json.Unmarshal(secretB, &secret); err != nil {
		panic(err)
	}

	return &secret
}

func (s *AnnounceKitService) CreateAuthToken(ctx context.Context, userClaims *JwtUserClaims) (string, error) {
	l := s.loggerProvider(ctx)

	if userClaims.ID == "" && userClaims.EMAIL == "" && userClaims.NAME == "" {
		return "", errors.New("expecting a non-falsy value on one of the following: id, email, name")
	}

	// Create a new token, specifying signing method and the claims
	tokenObj := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":        userClaims.ID,
		"email":     userClaims.EMAIL,
		"name":      userClaims.NAME,
		"iat":       time.Now().UTC().Unix(),
		"exp":       time.Now().UTC().Add(time.Hour * 72).Unix(),
		"expiresIn": "72 hours",
	})

	// Sign and get the complete encoded token as a string using the secret
	token, err := tokenObj.SignedString(s.jwtKeyB)
	if err != nil {
		l.Errorf("Failed to create announcekit jwt token\n Error: %v", err)
		return "", err
	}

	return token, nil
}

// GetChangeLogs gets AnnoucekitFeed from announcekit, filtered by the startDate
func (s *AnnounceKitService) GetChangeLogs(ctx context.Context, startDate time.Time) (AnnoucekitFeed, error) {
	var AKFeed AnnoucekitFeed

	req := http.Request{URL: "jsonfeed.json", ResponseType: &AKFeed}

	_, err := s.client.Get(ctx, &req)
	if err != nil {
		return AnnoucekitFeed{}, err
	}

	for i, item := range AKFeed.Items {
		AKFeed.Items[i].DateFormatted = item.DateModified.Format("January 2, 2006")
		// annoucekit jsonfeed is sorted by DateModified, newest update first
		if item.DateModified.Before(startDate) {
			AKFeed.Items = AKFeed.Items[0:i]
			break
		}
	}

	return AKFeed, nil
}
