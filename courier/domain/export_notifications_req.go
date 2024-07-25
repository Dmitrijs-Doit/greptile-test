package domain

type ExportNotificationsReq struct {
	StartDate       *string        `json:"startDate"`
	NotificationIDs []Notification `json:"notificationIds" binding:"required"`
}

func (req ExportNotificationsReq) Validate() error {
	for _, notificationID := range req.NotificationIDs {
		if err := validateNotificationID(notificationID); err != nil {
			return err
		}
	}

	return nil
}
