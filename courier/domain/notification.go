package domain

type Notification string

// When adding a new notification here, also add it to the validation in
//server/services/scheduled-tasks/courier/domain/export_notifications_req.go

const (
	DailyWeeklyDigestNotification       Notification = "9V4K6MQHZGMSM2GFK54HS3K2TNMY"
	NoClusterOnboardedNotification      Notification = "CCA2Y49NJDM6SWK5K4JF10JF1XXD"
	NotAllClustersOnboardedNotification Notification = "ZMKRM9AWTS4Y22Q4YQ3HZGYXVYEM"
	AtleastOneClusterFailedNotification Notification = "EPAFGEFE0GMCVHHM02HKFQJ4CV49"
)
