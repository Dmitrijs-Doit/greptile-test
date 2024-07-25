package common

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
)

func GetServiceAccountIDToken(ctx context.Context, audience string, secret []byte) (*oauth2.Token, error) {
	ts, err := GetServiceAccountTokenSource(ctx, audience, secret)
	if err != nil {
		return nil, err
	}

	token, err := ts.Token()
	if err != nil {
		return nil, err
	}

	return token, nil
}

func GetServiceAccountTokenSource(ctx context.Context, audience string, secret []byte) (oauth2.TokenSource, error) {
	option := idtoken.WithCredentialsJSON(secret)

	ts, err := idtoken.NewTokenSource(ctx, audience, option)
	if err != nil {
		return nil, err
	}

	return ts, err
}
