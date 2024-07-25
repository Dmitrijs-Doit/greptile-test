package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/gin-gonic/gin"
)

type DiscoveryTileDescriptor struct {
	Name        string                   `firestore:"name"`
	Title       string                   `firestore:"title"`
	Description string                   `firestore:"description"`
	Color       string                   `firestore:"color"`
	LinkLabel   string                   `firestore:"linkLabel"`
	Target      string                   `firestore:"target"`
	SortOrder   int16                    `firestore:"sortOrder"`
	TileAge     int16                    `firestore:"tileAge,omitempty"`
	Permissions []*firestore.DocumentRef `firestore:"permissions,omitempty"`
}

func PopulateDiscoveryTiles(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)

	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	discoveryTiles := []DiscoveryTileDescriptor{
		{
			Name:        "sign-up-to-live-and-on-demand-sessions",
			Title:       "Sign up to\nLive and on-demand sessions",
			Description: "Discover what you need to know to get started using the DoiT technology portfolio now.",
			Color:       "red",
			LinkLabel:   "Register now",
			Target:      "https://app.livestorm.co/doit",
			SortOrder:   0,
		},
		{
			Name:        "complete-our-onboarding-survey",
			Title:       "Complete our\nOnboarding survey",
			Description: "Please fill out this form so we can tailor your onboarding experience to your needs.",
			Color:       "blue",
			LinkLabel:   "Complete our short survey",
			Target:      "https://docs.google.com/forms/d/1PvfjN1zdYqIZiXy0skQ1FwK0JVpGjgGT5mS78hmJ7u4/viewform",
			SortOrder:   1,
			TileAge:     90,
		},
		{
			Name:        "understanding-your-doit-invoice",
			Title:       "Understanding\nyour DoiT invoice",
			Description: "View our documentation and FAQs to familiarise yourself with the different items on your invoice.",
			Color:       "navy",
			LinkLabel:   "Learn more",
			Target:      "https://help.doit.com/docs/billing/invoices-and-payments/understand-your-invoice",
			SortOrder:   2,
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionInvoices)),
			},
		},
		{
			Name:        "explore-our-cloud-training-services",
			Title:       "Explore our\ncloud training services",
			Description: "Explore our training courses for Google Workspace, Google Cloud and AWS.",
			Color:       "purple",
			LinkLabel:   "Browse our training marketplace",
			Target:      "/customers/:customerId:/training",
			SortOrder:   3,
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionPerksViewer)),
			},
		},
		{
			Name:        "gain-deeper-analysis",
			Title:       "Gain deeper analysis of your\ncloud spend and optimization insights",
			Description: "Connecting your historic billing data will allow you to achieve even more with the DoiT console.",
			Color:       "teal",
			LinkLabel:   "Connect now",
			Target:      "/customers/:customerId:/assets/google-cloud",
			SortOrder:   4,
			Permissions: []*firestore.DocumentRef{
				fs.Collection("permissions").Doc(string(common.PermissionAttributionsManager)),
			},
		},
		{
			Name:        "take-a-tour-of-the-doit-console",
			Title:       "Take a tour of the DoiT console",
			Description: "Get a full view of the products available to you and how they help manage and optimize your cloud spend.",
			Color:       "red",
			LinkLabel:   "Start the tour",
			Target:      "https://doit-international.navattic.com/c1de03pu",
			SortOrder:   5,
		},
		{
			Name:        "getting-started-with-attributions",
			Title:       "Getting started with Attributions",
			Description: "Learn how to organize and allocate your costs to provide the foundation for your analytics and optimization efforts.",
			Color:       "blue",
			LinkLabel:   "Learn more",
			Target:      "https://www.doit.com/map-cloud-costs-to-your-teams-environments-and-more-with-attributions/",
			SortOrder:   6,
		},
		{
			Name:        "learn-from-the-experts",
			Title:       "Learn from the experts",
			Description: "Listen and subscribe to the Cloud Masters podcast, featuring advice from DoiT's internal experts and customers.",
			Color:       "navy",
			LinkLabel:   "Explore the podcast",
			Target:      "https://www.youtube.com/playlist?list=PLEBxNMZ7Mu39fMIYhr5fdq8S-UrvTpEss",
			SortOrder:   7,
		},
	}

	for _, discoveryTile := range discoveryTiles {
		_, err := fs.Collection("dashboards").Doc("home").Collection("discoveryTiles").Doc(discoveryTile.Name).Set(ctx, discoveryTile)
		if err != nil {
			return []error{err}
		}
	}

	return nil
}
