package googleclouddirect

import (
	"context"
	"errors"
)

func (s *AssetService) Delete(ctx context.Context, id string) error {
	fs := s.conn.Firestore(ctx)

	assetRef := fs.Collection("assets").Doc(id)

	_, err := assetRef.Delete(ctx)
	if err != nil {
		return errors.New("delete has failed")
	}

	return nil
}
