package assets

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/firebase"
)

const (
	PlanAnnual   string = "ANNUAL"
	PlanFlexible string = "FLEXIBLE"

	GSuiteAssetPrefix string = "g-suite-"
)

type AssetSettingsLog struct {
	Email     string    `firestore:"email"`
	Settings  *Settings `firestore:"settings"`
	Timestamp time.Time `firestore:"timestamp,serverTimestamp"`
}

type AssetSettings struct {
	AssetType string                 `firestore:"type"`
	Customer  *firestore.DocumentRef `firestore:"customer"`
	Entity    *firestore.DocumentRef `firestore:"entity"`
	Contract  *firestore.DocumentRef `firestore:"contract"`
	Bucket    *firestore.DocumentRef `firestore:"bucket"`
	Settings  *Settings              `firestore:"settings"`
	Tags      []string               `firestore:"tags"`
}

type Settings struct {
	Plan     *SubscriptionPlan `firestore:"plan,omitempty"`
	Payment  string            `firestore:"payment,omitempty"`
	Currency string            `firestore:"currency,omitempty"`
}

type SubscriptionPlan struct {
	CommitmentInterval *SubscriptionPlanCommitmentInterval `firestore:"commitmentInterval,omitempty"`
	IsCommitmentPlan   bool                                `firestore:"isCommitmentPlan"`
	PlanName           string                              `firestore:"planName"`
}

type SubscriptionPlanCommitmentInterval struct {
	EndTime   int64 `firestore:"endTime,omitempty"`
	StartTime int64 `firestore:"startTime,omitempty"`
}

func (s *AssetService) UpdateAssetSettings(ctx context.Context, assetID string, email string, newSettings *Settings) error {
	fs := s.conn.Firestore(ctx)
	wb := firebase.NewAutomaticWriteBatch(fs, 500)

	err := s.updateAsset(ctx, wb, assetID, newSettings)
	if err != nil {
		return err
	}

	currSettings, err := s.updateAssetSettings(ctx, wb, assetID, email, newSettings)
	if err != nil {
		return err
	}

	// Currently we only update the inventory for GSuite assets when the plan is ANNUAL
	if newSettings != nil && currSettings.AssetType == common.Assets.GSuite &&
		newSettings.Plan != nil && newSettings.Plan.PlanName == PlanAnnual {
		if err := s.updateInventory(ctx, wb, assetID, currSettings, newSettings); err != nil {
			return err
		}
	}

	if errs := wb.Commit(ctx); len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (s *AssetService) updateAsset(ctx context.Context, wb *firebase.AutomaticWriteBatch, assetID string, settings *Settings) error {
	fs := s.conn.Firestore(ctx)

	wb.Update(fs.Collection("assets").Doc(assetID), []firestore.Update{
		{
			FieldPath: []string{"properties", "settings"},
			Value:     settings,
		},
	})

	return nil
}

func (s *AssetService) updateAssetSettings(ctx context.Context, wb *firebase.AutomaticWriteBatch, assetID string, email string, settings *Settings) (*AssetSettings, error) {
	fs := s.conn.Firestore(ctx)

	wb.Update(fs.Collection("assetSettings").Doc(assetID), []firestore.Update{
		{
			FieldPath: []string{"settings"},
			Value:     settings,
		},
	})

	assetSettingsDocRef := fs.Collection("assetSettings").Doc(assetID)
	wb.Create(assetSettingsDocRef.Collection("assetSettingsLogs").NewDoc(),
		AssetSettingsLog{
			Email:    email,
			Settings: settings,
		},
	)

	assetSettingsDocSnap, err := assetSettingsDocRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var assetSettings AssetSettings
	if err := assetSettingsDocSnap.DataTo(&assetSettings); err != nil {
		return nil, err
	}

	return &assetSettings, nil
}

func (s *AssetService) updateInventory(ctx context.Context, wb *firebase.AutomaticWriteBatch, assetID string, currAssetSettings *AssetSettings, newSettings *Settings) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	now := time.Now().Truncate(common.DayDuration)
	nowMillis := now.Unix() * 1000

	currSettings := currAssetSettings.Settings

	// skip inventory update is the commitment haven't started yet
	if nowMillis < newSettings.Plan.CommitmentInterval.StartTime {
		return nil
	}

	// skip inventory update if nothing changed (currency, payment and commitment interval dates are equal)
	if currSettings != nil && currSettings.Plan != nil && currSettings.Plan.CommitmentInterval != nil {
		if currSettings.Currency == newSettings.Currency &&
			currSettings.Payment == newSettings.Payment &&
			currSettings.Plan.CommitmentInterval.StartTime == newSettings.Plan.CommitmentInterval.StartTime &&
			currSettings.Plan.CommitmentInterval.EndTime == newSettings.Plan.CommitmentInterval.EndTime {
			return nil
		}
	}

	docSnaps, err := fs.CollectionGroup("inventory-g-suite").
		Where("subscriptionId", "==", assetID[len(GSuiteAssetPrefix):]).
		Where("date", "<", common.EpochMillisecondsToTime(newSettings.Plan.CommitmentInterval.EndTime)).
		Where("date", ">=", common.EpochMillisecondsToTime(newSettings.Plan.CommitmentInterval.StartTime)).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range docSnaps {
		wb.Update(docSnap.Ref, []firestore.Update{
			{
				FieldPath: []string{"settings"},
				Value:     newSettings,
			},
			{
				FieldPath: []string{"customer"},
				Value:     currAssetSettings.Customer,
			},
			{
				FieldPath: []string{"entity"},
				Value:     currAssetSettings.Entity,
			},
			{
				FieldPath: []string{"contract"},
				Value:     currAssetSettings.Contract,
			},
		})
		l.Debugf("Updating inventory item for %s", docSnap.Ref.ID)
	}

	return nil
}
