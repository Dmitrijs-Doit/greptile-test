package service

import (
	"context"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/microsoft/license/dal"
)

func (s *LicenseService) validateCreateOrder(ctx context.Context, props *CreateOrderProps, eDoc iface.DocumentSnapshot, cDoc iface.DocumentSnapshot) error {
	if !props.DoitEmployee {
		userID, ok := props.Claims["userId"]
		if !ok {
			return ErrUnauthorized
		}

		userDoc, err := s.dal.GetDoc(ctx, dal.UsersCollection, userID.(string))
		if err != nil {
			return ErrUnauthorized
		}

		var u common.User

		if err = userDoc.DataTo(&u); err != nil {
			return err
		}

		if u.Customer.Ref.ID != cDoc.Snapshot().Ref.ID {
			return ErrUnauthorized
		}

		if !u.HasLicenseManagePermission(ctx) {
			return ErrForbidden
		}
	}

	return nil
}
