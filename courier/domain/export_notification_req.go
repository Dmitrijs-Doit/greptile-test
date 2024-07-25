package domain

type ExportNotificationReq struct {
	StartDate      *string      `json:"startDate"`
	NotificationID Notification `json:"notificationId" binding:"required"`
}

func (req ExportNotificationReq) Validate() error {
	return validateNotificationID(req.NotificationID)
}

func validateNotificationID(notificationID Notification) error {
	switch notificationID {
	case DailyWeeklyDigestNotification,
		NoClusterOnboardedNotification,
		NotAllClustersOnboardedNotification,
		AtleastOneClusterFailedNotification:
		return nil
	}

	return ErrNotificationIDNotSupported
}
