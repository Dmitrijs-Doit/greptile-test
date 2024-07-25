package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
)

type WhenCollectionIsEmpty string

type QuickLinkCondition struct {
	WhenCollectionIsEmpty          WhenCollectionIsEmpty `firestore:"whenCollectionIsEmpty"`
	CollectionCustomerRefFieldName string                `firestore:"collectionCustomerRefFieldName,omitempty"`
	CollectionPath                 string                `firestore:"collectionPath"`
	CollectionFilters              string                `firestore:"collectionFilters,omitempty"`
}

type QuickLinkDescriptor struct {
	Name        string `firestore:"name"`
	Title       string `firestore:"title"`
	Description string `firestore:"description"`
	Icon        string `firestore:"icon"`
	Target      string `firestore:"target"`
	SortOrder   int16  `firestore:"sortOrder"`

	Permissions      []*firestore.DocumentRef `firestore:"permissions"`
	UserMaturityFrom int                      `firestore:"userMaturityFrom,omitempty"`
	UserMaturityTo   int                      `firestore:"userMaturityTo,omitempty"`
	Conditions       []*QuickLinkCondition    `firestore:"conditions,omitempty"`
}

const (
	WhenCollectionIsEmptyHide WhenCollectionIsEmpty = "hide"
	WhenCollectionIsEmptyShow WhenCollectionIsEmpty = "show"
)

func PopulateQuickLinks(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)

	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	quickLinks := []QuickLinkDescriptor{
		{
			Name:        "view-cloud-incidents",
			Title:       "View cloud incidents",
			Description: "Google Cloud and AWS known issues, performance and availability updates",
			Icon:        "incidents",
			Target:      "/customers/:customerId:/known-issues",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionIssuesViewer)),
			},
			SortOrder: 51,
		},
		{
			Name:        "view-and-manage-assets",
			Title:       "View and manage assets",
			Description: "Self service management of your cloud assets",
			Icon:        "cloud",
			Target:      "/customers/:customerId:/assets",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionAssetsManager)),
			},
			SortOrder:      52,
			UserMaturityTo: 21,
		},
		{
			Name:        "create-support-request",
			Title:       "Create a support request",
			Description: "Direct access to our best in-class Cloud Reliability Engineers (CREs)",
			Icon:        "ticket",
			Target:      "/customers/:customerId:/support/new",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionSupportRequester)),
			},
			SortOrder: 50,
		},
		{
			Name:           "manage-notifications",
			Title:          "Manage notifications",
			Description:    "Manage various notification types and select email digests for your account",
			Icon:           "notifications",
			Target:         "/customers/:customerId:/profile/:userId:/notifications",
			Permissions:    []*firestore.DocumentRef{},
			SortOrder:      20,
			UserMaturityTo: 21,
		},
		{
			Name:        "create-attribution",
			Title:       "Create an attribution",
			Description: "Group cloud resources based on your business aspects. i.e teams or projects",
			Icon:        "plus",
			Target:      "/customers/:customerId:/analytics/attributions/create",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionAttributionsManager)),
			},
			SortOrder:        13,
			UserMaturityFrom: 21,
		},
		{
			Name:        "create-report",
			Title:       "Create a report",
			Description: "Gain instant visibility into your Google Cloud and AWS costs",
			Icon:        "plus",
			Target:      "/customers/:customerId:/analytics/reports/create",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionCloudAnalytics)),
			},
			SortOrder:        11,
			UserMaturityFrom: 21,
		},
		{
			Name:        "create-attribution-group",
			Title:       "Create an attribution group",
			Description: "Distribute data for a particular metric among multiple resource groups",
			Icon:        "plus",
			Target:      "/customers/:customerId:/analytics/attribution-groups/create",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionAttributionsManager)),
			},
			SortOrder: 14,
		},
		{
			Name:        "create-alert",
			Title:       "Create an alert",
			Description: "Track various dimensions in your cloud environments",
			Icon:        "plus",
			Target:      "/customers/:customerId:/analytics/alerts",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionAttributionsManager)),
			},
			SortOrder:        15,
			UserMaturityFrom: 21,
		},
		{
			Name:        "view-cost-anomalies",
			Title:       "View cost anomalies",
			Description: "Monitoring spikes in your Google Cloud and AWS costs",
			Icon:        "insights",
			Target:      "/customers/:customerId:/anomaly",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionAnomaliesViewer)),
			},
			SortOrder:        45,
			UserMaturityFrom: 21,
		},
		{
			Name:        "connect-your-gcp-account",
			Title:       "Connect your GCP account",
			Description: "Access portfolio features such as BigQuery Lens and Sandboxes",
			Icon:        "plus",
			Target:      "/customers/:customerId:/settings/gcp",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionAttributionsManager)),
			},
			SortOrder: 0,
			Conditions: []*QuickLinkCondition{
				{
					WhenCollectionIsEmpty:          WhenCollectionIsEmptyHide,
					CollectionCustomerRefFieldName: "customer",
					CollectionPath:                 "assets",
					CollectionFilters:              "{\"type\":[\"g-suite\",\"google-cloud\",\"google-cloud-project\",\"google-cloud-direct\",\"google-cloud-standalone\"]}",
				},
				{
					WhenCollectionIsEmpty:          WhenCollectionIsEmptyShow,
					CollectionCustomerRefFieldName: "customer",
					CollectionPath:                 "/customers/:customerId:/cloudConnect",
					CollectionFilters:              "{\"cloudPlatform\":\"google-cloud\"}",
				},
			},
		},
		{
			Name:        "connect-historic-billing-data",
			Title:       "Connect historic billing data",
			Description: "Deeper analysis of your cloud spend and identify optimization opportunities",
			Icon:        "plus",
			Target:      "/customers/:customerId:/assets/google-cloud",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionAttributionsManager)),
			},
			SortOrder:      1,
			UserMaturityTo: 21,
		},
		{
			Name:        "invite-colleagues",
			Title:       "Invite colleagues",
			Description: "Manage the access of other users on behalf of the organization",
			Icon:        "person",
			Target:      "/customers/:customerId:/iam/users",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionUsersManager)),
			},
			SortOrder:      30,
			UserMaturityTo: 21,
		},
		{
			Name:        "set-up-ramp-plan",
			Title:       "Set up a ramp plan",
			Description: "Ramp plans can help you meet your spend-based commitment goals",
			Icon:        "rampPlan",
			Target:      "/customers/:customerId:/contracts/ramps",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionInvoices)),
			},
			SortOrder: 40,
			Conditions: []*QuickLinkCondition{
				{
					WhenCollectionIsEmpty:          WhenCollectionIsEmptyShow,
					CollectionCustomerRefFieldName: "customerRef",
					CollectionPath:                 "rampPlans",
				},
			},
		},
		{
			Name:        "display-ramp-progress",
			Title:       "Display ramp progress",
			Description: "View your ramp plan progress in detail",
			Icon:        "rampPlan",
			Target:      "/customers/:customerId:/contracts/ramps",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionInvoices)),
			},
			SortOrder: 40,
			Conditions: []*QuickLinkCondition{
				{
					WhenCollectionIsEmpty:          WhenCollectionIsEmptyHide,
					CollectionCustomerRefFieldName: "customerRef",
					CollectionPath:                 "rampPlans",
				},
			},
		},
		{
			Name:        "eks-onboarding",
			Title:       "Connect your EKS clusters",
			Description: "Start measuring your EKS clusters cost and usage and identify areas for improvement",
			Icon:        "plus",
			Target:      "/customers/:customerId:/eks-onboarding",
			SortOrder:   13,
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionSettings)),
			},
			Conditions: []*QuickLinkCondition{
				{
					WhenCollectionIsEmpty:          WhenCollectionIsEmptyHide,
					CollectionCustomerRefFieldName: "customerId",
					CollectionPath:                 "/integrations/k8s-metrics/eks",
				},
			},
		},
		{
			Name:        "create-budget",
			Title:       "Create a budget",
			Description: "Track actual cloud spend against planned spend to avoid billing surprises",
			Icon:        "plus",
			Target:      "/customers/:customerId:/analytics/budgets",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionBudgetsManager)),
			},
			SortOrder:        12,
			UserMaturityFrom: 21,
		},
		{
			Name:        "guided-experience",
			Title:       "Start allocating costs",
			Description: "Break down your cloud costs and usage in ways that are meaningful to your business",
			Icon:        "plus",
			Target:      "#",
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionAttributionsManager)),
			},
			SortOrder: 16,
		},
	}

	for _, quickLink := range quickLinks {
		_, err := fs.Collection("dashboards").Doc("home").Collection("quickLinks").Doc(quickLink.Name).Set(ctx, quickLink)
		if err != nil {
			return []error{err}
		}
	}

	return nil
}
