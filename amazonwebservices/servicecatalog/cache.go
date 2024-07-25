package servicecatalog

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CacheDoc struct {
	PortfolioID string `firestore:"portfolioId"`
}

type CacheKey struct {
	Name      string
	AccountID string
	Region    string
}

var ErrCacheMiss = errors.New("cache miss")

type Cache interface {
	Get(ctx context.Context, key CacheKey) (string, error)
	Set(ctx context.Context, key CacheKey, value string) error
	Del(ctx context.Context, key CacheKey) error
}

type FSCache struct {
	fsProvider fsProvider
	colPath    string
}

func (c FSCache) formatKey(key CacheKey) string {
	return fmt.Sprintf("%s-%s-%s", key.Name, key.AccountID, key.Region)
}

func (c FSCache) Get(ctx context.Context, key CacheKey) (string, error) {
	col := c.fsProvider(ctx).Collection(c.colPath)

	cacheDoc, err := col.Doc(c.formatKey(key)).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return "", ErrCacheMiss
		}

		return "", err
	}

	var cache CacheDoc
	if err := cacheDoc.DataTo(&cache); err != nil {
		return "", err
	}

	return cache.PortfolioID, nil
}

func (c FSCache) Set(ctx context.Context, key CacheKey, value string) error {
	col := c.fsProvider(ctx).Collection(c.colPath)
	_, err := col.Doc(c.formatKey(key)).Set(ctx, CacheDoc{
		PortfolioID: value,
	})

	return err
}

func (c FSCache) Del(ctx context.Context, key CacheKey) error {
	col := c.fsProvider(ctx).Collection(c.colPath)

	_, err := col.Doc(c.formatKey(key)).Delete(ctx)

	return err
}
