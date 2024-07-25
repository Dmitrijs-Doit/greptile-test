package organizations

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (s *OrgsIAMService) handleDeleteOrgs(ctx context.Context, fs *firestore.Client, req *RemoveIAMOrgsRequest) error {
	for _, orgID := range req.OrgIDs {
		if orgID == RootOrgID {
			return errors.New("root organization may not be deleted")
		}
	}

	bulkWriter := fs.BulkWriter(ctx)

	if err := s.DeleteOrgsFromFirestore(ctx, fs, bulkWriter, req); err != nil {
		return err
	}

	if err := s.DeleteOrgsFromUsers(ctx, fs, bulkWriter, req); err != nil {
		return err
	}

	if err := s.DeleteOrgsReports(ctx, fs, bulkWriter, req); err != nil {
		return err
	}

	if err := s.DeleteOrgsReportWidgets(ctx, fs, bulkWriter, req); err != nil {
		return err
	}

	bulkWriter.End()

	return nil
}

func (s *OrgsIAMService) DeleteOrgsFromFirestore(
	ctx context.Context,
	fs *firestore.Client,
	bulkWriter *firestore.BulkWriter,
	req *RemoveIAMOrgsRequest,
) error {
	documentHandler := doitFirestore.DocumentHandler{}

	for _, orgID := range req.OrgIDs {
		orgRef := fs.Collection("customers").Doc(req.CustomerID).Collection("customerOrgs").Doc(orgID)

		if orgID == RootOrgID {
			continue
		}

		if err := documentHandler.DeleteDocAndSubCollections(ctx, orgRef, bulkWriter); err != nil {
			return err
		}
	}

	return nil
}

func (s *OrgsIAMService) DeleteOrgsFromUsers(
	ctx context.Context,
	fs *firestore.Client,
	bulkWriter *firestore.BulkWriter,
	req *RemoveIAMOrgsRequest,
) error {
	orgsCollection := fs.Collection("customers").Doc(req.CustomerID).Collection("customerOrgs")
	for _, orgID := range req.OrgIDs {
		orgRef := orgsCollection.Doc(orgID)

		userSnaps, err := fs.Collection("users").Where("organizations", "array-contains", orgRef).Select("organizations").Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		inviteSnaps, err := fs.Collection("invites").Where("organizations", "array-contains", orgRef).Select("organizations").Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		for _, userSnap := range append(userSnaps, inviteSnaps...) {
			_, err := bulkWriter.Update(userSnap.Ref, []firestore.Update{{Path: "organizations", Value: firestore.ArrayRemove(orgRef)}})
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *OrgsIAMService) GetCurrentUser(ctx context.Context, fs *firestore.Client, userID string) (*common.User, error) {
	userRef := fs.Collection("users").Doc(userID)

	user, err := common.GetUser(ctx, userRef)
	if err != nil {
		return nil, ErrUserNotFound
	}

	return user, nil
}

func (s *OrgsIAMService) DeleteOrgsReports(
	ctx context.Context,
	fs *firestore.Client,
	bulkWriter *firestore.BulkWriter,
	req *RemoveIAMOrgsRequest,
) error {
	for _, orgID := range req.OrgIDs {
		orgRef := fs.Collection("customers").Doc(req.CustomerID).Collection("customerOrgs").Doc(orgID)

		docSnaps, err := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("savedReports").Where("organization", "==", orgRef).Select("organization").Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		for _, docSnap := range docSnaps {
			if _, err := bulkWriter.Delete(docSnap.Ref); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *OrgsIAMService) DeleteOrgsReportWidgets(
	ctx context.Context,
	fs *firestore.Client,
	bulkWriter *firestore.BulkWriter,
	req *RemoveIAMOrgsRequest,
) error {
	for _, orgID := range req.OrgIDs {
		orgRef := fs.Collection("customers").Doc(req.CustomerID).Collection("customerOrgs").Doc(orgID)

		docSnaps, err := fs.Collection("cloudAnalytics").Doc("widgets").Collection("cloudAnalyticsWidgets").Where("organization", "==", orgRef).Select().Documents(ctx).GetAll()
		if err != nil {
			return err
		}

		for _, docSnap := range docSnaps {
			if _, err := bulkWriter.Delete(docSnap.Ref); err != nil {
				return err
			}
		}
	}

	return nil
}
