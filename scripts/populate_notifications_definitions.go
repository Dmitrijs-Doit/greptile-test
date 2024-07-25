package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
)

type NotificationAudience string
type NotificationProvider string
type NotificationCommonFilter string
type NotificationValue int8

const (
	NotificationAudienceUser    NotificationAudience = "user"
	NotificationAudienceCompany NotificationAudience = "company"

	NotificationProviderEmail NotificationProvider = "EMAIL"
	NotificationProviderSlack NotificationProvider = "SLACK"

	NotificationCommonFilterAttributions NotificationCommonFilter = "ATTRIBUTIONS"

	NotificationNewInvoices           NotificationValue = 1
	NotificationCloudCostAnomalies    NotificationValue = 2
	NotificationPaymentDueOverdue     NotificationValue = 3
	NotificationCreditsUtilization    NotificationValue = 4
	NotificationCloudQuotaUtilization NotificationValue = 5
	NotificationCloudIncidents        NotificationValue = 6
	NotificationDailyDigests          NotificationValue = 7
	NotificationMonthlyDigests        NotificationValue = 8
	NotificationWeeklyDigests         NotificationValue = 9

	CloudCostAnomaliesNotificationGroup = "cloud-cost-anomalies"
	CloudGovernanceNotificationGroup    = "cloud-governance"
)

type NotificationGroupDescriptor struct {
	Name        string `firestore:"name"`
	Title       string `firestore:"title"`
	Description string `firestore:"description"`
}

type NotificationCommonFilterDescriptor struct {
	Type     NotificationCommonFilter `firestore:"type"`
	Required bool                     `firestore:"required"`
}

type NotificationDescriptor struct {
	Name        string            `firestore:"name"`
	Description string            `firestore:"description,omitempty"`
	Order       int8              `firestore:"order"`
	Value       NotificationValue `firestore:"value"`
	// Permissions        []*firestore.DocumentRef   `firestore:"permissions"`
	Group              string                               `firestore:"group,omitempty"`
	Audience           []NotificationAudience               `firestore:"audience"`
	SupportedProviders []NotificationProvider               `firestore:"supportedProviders"`
	CommonFilters      []NotificationCommonFilterDescriptor `firestore:"commonFilters"`
}

type NotificationPermission struct {
	User    []*firestore.DocumentRef `firestore:"user"`
	Company []*firestore.DocumentRef `firestore:"company"`
}

type NotificationsDoc struct {
	Notifications           []*NotificationDescriptor      `firestore:"notifications"`
	NotificationGroups      []*NotificationGroupDescriptor `firestore:"notificationGroups"`
	NotificationPermissions [10]NotificationPermission     `firestore:"notificationPermissions"`
}

func PopulateNotificationsDefinitions(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)

	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	var notificationPermissions = [10]NotificationPermission{}

	notificationPermissions[NotificationNewInvoices] = NotificationPermission{
		User: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionInvoices)),
		},
		Company: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionSettings)),
		},
	}

	notificationPermissions[NotificationCloudCostAnomalies] = NotificationPermission{
		User: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionCloudAnalytics)),
		},
		Company: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionSettings)),
		},
	}

	notificationPermissions[NotificationPaymentDueOverdue] = NotificationPermission{
		User: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionInvoices)),
		},
		Company: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionSettings)),
		},
	}

	notificationPermissions[NotificationCreditsUtilization] = NotificationPermission{
		User: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionInvoices)),
		},
		Company: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionSettings)),
		},
	}

	notificationPermissions[NotificationCloudQuotaUtilization] = NotificationPermission{
		User: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionCloudAnalytics)),
		},
		Company: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionSettings)),
		},
	}

	notificationPermissions[NotificationCloudIncidents] = NotificationPermission{
		User: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionIssuesViewer)),
		},
		Company: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionSettings)),
		},
	}

	notificationPermissions[NotificationDailyDigests] = NotificationPermission{
		User: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionCloudAnalytics)),
		},
		Company: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionSettings)),
		},
	}

	notificationPermissions[NotificationMonthlyDigests] = NotificationPermission{
		User: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionCloudAnalytics)),
		},
		Company: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionSettings)),
		},
	}

	notificationPermissions[NotificationWeeklyDigests] = NotificationPermission{
		User: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionCloudAnalytics)),
		},
		Company: []*firestore.DocumentRef{
			fs.Collection("permissions").Doc(string(common.PermissionSettings)),
		},
	}

	groups := []*NotificationGroupDescriptor{
		{
			Name:  "digests",
			Title: "Digests",
			// Description: "Receive digests for your account",
		},
		{
			Name:  "invoices",
			Title: "Invoices",
		},
		{
			Name:  CloudCostAnomaliesNotificationGroup,
			Title: "Cloud cost anomalies",
		},
		{
			Name:  CloudGovernanceNotificationGroup,
			Title: "Cloud governance",
		},
	}

	notifications := []*NotificationDescriptor{
		{
			Name:        "New invoices",
			Description: "We'll notify you every time there is a new invoice or adjustment",
			Order:       1,
			Value:       NotificationNewInvoices,
			Audience: []NotificationAudience{
				NotificationAudienceUser,
			},
			SupportedProviders: []NotificationProvider{
				NotificationProviderEmail,
			},
			Group: "invoices",
		},
		{
			Name:        "Cost anomaly detection",
			Description: "We'll alert you when your cloud costs exceed the predicted normal spending behavior",
			Order:       6,
			Value:       NotificationCloudCostAnomalies,
			Audience: []NotificationAudience{
				NotificationAudienceUser,
				NotificationAudienceCompany,
			},
			SupportedProviders: []NotificationProvider{
				NotificationProviderEmail,
				NotificationProviderSlack,
			},
			CommonFilters: []NotificationCommonFilterDescriptor{
				{
					Type:     NotificationCommonFilterAttributions,
					Required: false,
				},
			},
			Group: CloudCostAnomaliesNotificationGroup,
		},
		{
			Name:        "Due or overdue payments",
			Description: "Get automated updates when invoices are due and overdue.",
			Order:       2,
			Value:       NotificationPaymentDueOverdue,
			Audience: []NotificationAudience{
				NotificationAudienceUser,
			},
			SupportedProviders: []NotificationProvider{
				NotificationProviderEmail,
			},
			Group: "invoices",
		},
		{
			Name:        "Credit utilization",
			Description: "We'll let you know on 75% utilization of your Google Cloud or AWS credits and then again when they are fully exhausted",
			Order:       3,
			Value:       NotificationCreditsUtilization,
			Audience: []NotificationAudience{
				NotificationAudienceUser,
			},
			SupportedProviders: []NotificationProvider{
				NotificationProviderEmail,
			},
			Group: CloudGovernanceNotificationGroup,
		},
		{
			Name:        "Quota utilization",
			Description: "Be notified every time one of your Google Cloud or AWS service quotas utilization is over 50%",
			Order:       4,
			Value:       NotificationCloudQuotaUtilization,
			Audience: []NotificationAudience{
				NotificationAudienceUser,
				NotificationAudienceCompany,
			},
			SupportedProviders: []NotificationProvider{
				NotificationProviderEmail,
				NotificationProviderSlack,
			},
			Group: CloudGovernanceNotificationGroup,
		},
		{
			Name:        "Cloud incidents",
			Description: "Be aware of the known infrastructure issues with Google Cloud or Amazon Web Services ahead of time",
			Order:       5,
			Value:       NotificationCloudIncidents,
			Audience: []NotificationAudience{
				NotificationAudienceUser,
				NotificationAudienceCompany,
			},
			SupportedProviders: []NotificationProvider{
				NotificationProviderEmail,
				NotificationProviderSlack,
			},
			Group: CloudGovernanceNotificationGroup,
		},
		{
			Name:  "Daily digests",
			Order: 7,
			Value: NotificationDailyDigests,
			Audience: []NotificationAudience{
				NotificationAudienceUser,
				NotificationAudienceCompany,
			},
			SupportedProviders: []NotificationProvider{
				NotificationProviderEmail,
				NotificationProviderSlack,
			},
			Group: "digests",
			CommonFilters: []NotificationCommonFilterDescriptor{
				{
					Type:     NotificationCommonFilterAttributions,
					Required: true,
				},
			},
		},
		{
			Name:  "Monthly digests",
			Order: 9,
			Value: NotificationMonthlyDigests,
			Audience: []NotificationAudience{
				NotificationAudienceUser,
			},
			SupportedProviders: []NotificationProvider{
				NotificationProviderEmail,
			},
			Group: "digests",
		},
		{
			Name:  "Weekly digests",
			Order: 8,
			Value: NotificationWeeklyDigests,
			Audience: []NotificationAudience{
				NotificationAudienceUser,
			},
			SupportedProviders: []NotificationProvider{
				NotificationProviderEmail,
			},
			Group: "digests",
			CommonFilters: []NotificationCommonFilterDescriptor{
				{
					Type:     NotificationCommonFilterAttributions,
					Required: true,
				},
			},
		},
	}

	_, err = fs.Collection("app").Doc("notifications").Set(ctx, NotificationsDoc{
		Notifications:           notifications,
		NotificationGroups:      groups,
		NotificationPermissions: notificationPermissions,
	})
	if err != nil {
		return []error{err}
	}

	return nil
}
