package service

import (
	"strconv"
)

type AssetWithName struct {
	assetID   string
	assetName string
}

func mapToToken(tokens []string, numericID string) (string, error) {
	numericValue, err := strconv.Atoi(numericID)
	if err != nil {
		return "", err
	}

	return tokens[numericValue%len(tokens)], nil
}

func generateAssetTagFromNumericID(numericID string) (string, error) {
	tags, err := mapToToken(tags, numericID)
	if err != nil {
		return "", err
	}

	return tags, nil
}
