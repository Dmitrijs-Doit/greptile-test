package dal

type reportField = string

const (
	reportFieldCollaborators reportField = "collaborators"
	reportFieldPublic        reportField = "public"
	reportFieldConfig        reportField = "config"
	reportFieldCustomer      reportField = "customer"
	reportFieldDescription   reportField = "description"
	reportFieldDraft         reportField = "draft"
	reportFieldName          reportField = "name"
	reportFieldOrganization  reportField = "organization"
	reportFieldSchedule      reportField = "schedule"
	reportFieldTimeModified  reportField = "timeModified"
	reportFieldType          reportField = "type"
	reportFieldWidgetEnabled reportField = "widgetEnabled"
	reportFieldHidden        reportField = "hidden"
	reportFieldCloud         reportField = "cloud"
	reportFieldLabels        reportField = "labels"
)
