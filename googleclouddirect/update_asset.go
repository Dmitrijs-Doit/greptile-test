package googleclouddirect

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
)

type AssetFindDetails struct {
	CustomerId string `json:"customerId" binding:"required"`
	AssetId    string `json:"assetId" binding:"required"`
}

type AssetDataToUpdate struct {
	Dataset string `json:"dataset"`
	Project string `json:"project"`
}

func (s *AssetService) Update(ctx context.Context, id string, updateData *AssetDataToUpdate) error {
	fs := s.conn.Firestore(ctx)
	l := s.loggerProvider(ctx)

	ref := fs.Collection("assets").Doc(id)

	snap, err := ref.Get(ctx)
	if err != nil {
		return errors.New("error getting asset")
	}

	var data GoogleCloudBillingAsset

	if err := snap.DataTo(&data); err != nil {
		l.Error("unable to cast snap to asset struct")
		return errors.New("error getting asset")
	}

	tables := data.Tables
	if len(tables) != 1 {
		l.Error("invalid length of tables property on asset")
		return errors.New("update has failed")
	}

	table := tables[0]

	if updateData.Dataset != "" {
		ok, _ := MatchDataset(updateData.Dataset)
		if !ok {
			return errors.New("dataset format is incorrect")
		}

		table.Dataset = updateData.Dataset
	}

	if updateData.Project != "" {
		ok, _ := MatchProject(updateData.Project)
		if !ok {
			return errors.New("project format is incorrect")
		}

		table.Project = updateData.Project
	}

	_, err = ref.Update(ctx, []firestore.Update{
		{
			Path:  "tables",
			Value: []Table{table},
		},
	})
	if err != nil {
		return errors.New("update has failed")
	}

	return nil
}
