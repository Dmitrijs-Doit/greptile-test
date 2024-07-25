package hubspot

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

// ContactSearchRes contacts search response
type ContactSearchRes struct {
	Total   float64   `json:"total"`
	Results []Contact `json:"results"`
}

type Contact struct {
	ID         string             `json:"id"`
	Properties *ContactProperties `json:"properties"`
}

type ContactProperties struct {
	JobFunction    string `json:"job_function"`
	Email          string `json:"email"`
	FirstName      string `json:"firstname"`
	LastName       string `json:"lastname"`
	Origin         bool   `json:"cmp_origin"`
	FirstLogin     int64  `json:"cmp_first_login,omitempty"`
	LastLogin      int64  `json:"cmp_last_login,omitempty"`
	CMPRoles       string `json:"cmp_roles,omitempty"`
	CMPPermissions string `json:"cmp_permissions,omitempty"`
}

// updateHsContact
type updateHsContact struct {
	Properties ContactProperties `json:"properties"`
}

var userRoleMap = map[int64]string{
	0: "", // Empty
	1: "Software / Ops Engineer",
	2: "Finance / Accounting",
	3: "Operations",
	4: "Sales / Marketing",
	5: "Legal / Purchasing",
	6: "Founder",
	7: "Management",
}

var defaultContactsRet = []string{"firstname", "lastname", "job_function", "email", "hs_object_id"}

func contactCreateOrUpdateError(err error, l logger.ILogger) (bool, error) {
	errStr := err.Error()

	l.Warningf("contactCreateOrUpdateError error: %s", errStr)

	switch {
	case strings.Contains(errStr, "VALIDATION_ERROR"):
		l.Warningf("Contacts.Update VALIDATION_ERROR: %s", err)
		return false, nil
	case strings.Contains(errStr, "CONTAINS_URL"):
		l.Warningf("Contacts.Update CONTAINS_URL: %s", err)
		return true, nil
	default:
		return false, err
	}
}

// SyncContactsWorker hubspot sync a single customer contacts
func (s *HubspotService) SyncContactsWorker(ctx context.Context, customerID string) error {
	l := s.Logger(ctx)

	hubspotService, err := NewService(ctx)
	if err != nil {
		return err
	}

	auth, err := s.TenantService.GetTenantAuthClientByCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	customerRef := s.Firestore(ctx).Collection("customers").Doc(customerID)

	isAssociatedToCompany, err := checkAssociation(ctx, hubspotService, customerRef)
	if err != nil {
		l.Debugf("checkAssociation: %s", err.Error())
		return err
	}

	if !isAssociatedToCompany {
		l.Debugf("checkAssociation: customer %s was not found in hubspot", customerID)
		return err
	}

	docSnaps, err := s.Firestore(ctx).Collection("users").
		WherePath([]string{"customer", "ref"}, "==", customerRef).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range docSnaps {
		var user common.User
		if err := docSnap.DataTo(&user); err != nil {
			return err
		}

		// set user role to string
		switch t := user.Role.(type) {
		case int64:
			user.Role = userRoleMap[t]
		default:
			user.Role = ""
		}

		userRecord, err := auth.GetUserByEmail(ctx, user.Email)
		if err != nil {
			l.Warningf("auth.GetUserByEmail %s: %s", user.Email, err.Error())
			continue
		}

		userUpdatePayload := ContactProperties{
			FirstName:   strings.Title(user.FirstName),
			LastName:    strings.Title(user.LastName),
			Email:       user.Email,
			JobFunction: user.Role.(string),
			Origin:      true,
		}

		if userRecord.UserMetadata != nil {
			// Hubspot accepts only unix milliseconds at midnight for a given day
			//  => .UTC().Truncate(24*time.Hour).Unix() * 1000
			if userRecord.UserMetadata.CreationTimestamp > 0 {
				userUpdatePayload.FirstLogin = common.EpochMillisecondsToTime(userRecord.UserMetadata.CreationTimestamp).UTC().Truncate(24*time.Hour).Unix() * 1000
			}

			if userRecord.UserMetadata.LastLogInTimestamp > 0 {
				userUpdatePayload.LastLogin = common.EpochMillisecondsToTime(userRecord.UserMetadata.LastLogInTimestamp).UTC().Truncate(24*time.Hour).Unix() * 1000
			}
		}

		userUpdatePayload.CMPRoles, userUpdatePayload.CMPPermissions, err = getUserRolesAndPermissions(ctx, &user)
		if err != nil {
			l.Debugf("getUserRolesAndPermissions: %s", err.Error())
			return err
		}

		contact, err := searchCompareContacts(ctx, hubspotService, &userUpdatePayload)
		if err != nil {
			l.Debugf("searchCompareContacts: %s", err.Error())
			return err
		}

		if contact == nil {
			continue
		}

		if len(contact.ID) == 0 {
			// contact does not exists
			if err := hubspotService.Contacts.Create(ctx, updateHsContact{Properties: userUpdatePayload}); err != nil {
				if ok, innerErr := contactCreateOrUpdateError(err, l); !ok {
					return innerErr
				}
			}
		} else {
			// contact needs to be updated
			if err := hubspotService.Contacts.Update(ctx, updateHsContact{Properties: userUpdatePayload}, contact.ID); err != nil {
				if ok, innerErr := contactCreateOrUpdateError(err, l); !ok {
					return innerErr
				}
			}
		}
	}

	return nil
}

// Search firebase and then search hubspot for properties retreived
func searchCompareContacts(ctx context.Context, hubspotService *Service, user *ContactProperties) (*Contact, error) {
	body := hsReq{
		Properties: defaultContactsRet,
		FilterGroups: []Filters{
			{
				Filters: []Filter{
					{
						PropertyName: "email",
						Operator:     FilterOperatorEquals,
						Value:        user.Email,
					},
				},
			},
			{
				Filters: []Filter{
					{
						PropertyName: "hs_additional_emails",
						Operator:     FilterOperatorEquals,
						Value:        user.Email,
					},
				},
			},
		},
		Sorts: sorts,
	}

	resp, err := hubspotService.Search(ctx, body, "contacts")
	if err != nil {
		return nil, err
	}

	var res ContactSearchRes

	err = json.Unmarshal(resp, &res)
	if err != nil {
		return nil, err
	}

	if res.Total == 0 {
		return &Contact{
			ID:         "",
			Properties: nil,
		}, nil
	}

	if res.Total == 1 {
		contactProps := res.Results[0].Properties
		contactID := res.Results[0].ID

		// only update the contact when the hubspot contact's primary email is equal to the CMP user email.
		// this is required because in hubspot some emails are consolidated into a single "contact"
		// while in CMP there can be separate users.
		// we do not want to update the name/role in hubspot for every single CMP user that match.
		if contactProps.Email != user.Email {
			return nil, nil
		}

		if user.FirstName == "" {
			user.FirstName = contactProps.FirstName
		}

		if user.LastName == "" {
			user.LastName = contactProps.LastName
		}

		firstNameEq := strings.EqualFold(contactProps.FirstName, user.FirstName)
		lastNameEq := strings.EqualFold(contactProps.LastName, user.LastName)
		roleEq := contactProps.JobFunction == user.JobFunction
		firstLoginEq := contactProps.FirstLogin == user.FirstLogin
		lastLoginEq := contactProps.LastLogin == user.LastLogin
		permissionEq := slice.UnorderedSeparatedStringsComp(contactProps.CMPPermissions, user.CMPPermissions, HubspotArraySeparator)
		cmpRoleEq := slice.UnorderedSeparatedStringsComp(contactProps.CMPRoles, user.CMPRoles, HubspotCustomRoleSeparator)

		if firstNameEq && lastNameEq && roleEq && firstLoginEq && lastLoginEq && permissionEq && cmpRoleEq {
			return nil, nil
		}

		return &Contact{
			ID:         contactID,
			Properties: contactProps,
		}, nil
	}

	return nil, errors.New("more than one contact found")
}

func checkAssociation(ctx context.Context, hubspotService *Service, customerRef *firestore.DocumentRef) (bool, error) {
	company, err := queryHS(ctx, hubspotService, hsReq{
		Properties: defaultCompanyRet,
		FilterGroups: []Filters{
			{
				Filters: []Filter{
					{
						PropertyName: "cmp_external_id",
						Operator:     FilterOperatorEquals,
						Value:        customerRef.ID,
					},
				},
			},
		},
		Sorts: sorts,
	}, customerRef.ID)
	if err != nil {
		return false, err
	}

	if company != nil {
		return true, nil
	}

	return false, nil
}

func getUserRolesAndPermissions(ctx context.Context, user *common.User) (string, string, error) {
	// check if using new role based permissions
	if user.Roles != nil && len(user.Roles) > 0 {
		roles := make([]string, 0, len(user.Roles))
		// unique permissions across all roles
		permissionSet := make(map[string]struct{})

		for _, role := range user.Roles {
			roleDocSnap, err := role.Get(ctx)
			if err != nil {
				return "", "", err
			}

			var r common.Role
			if err := roleDocSnap.DataTo(&r); err != nil {
				return "", "", err
			}

			roles = append(roles, r.Name)

			for _, p := range r.Permissions {
				permissionSet[p.ID] = struct{}{}
			}
		}

		permissions := make([]string, 0, len(permissionSet))
		for k := range permissionSet {
			permissions = append(permissions, k)
		}

		return strings.Join(roles, HubspotCustomRoleSeparator), strings.Join(permissions, HubspotArraySeparator), nil
	}
	// legacy permissions with no role attached
	return "", strings.Join(user.Permissions, HubspotArraySeparator), nil
}
