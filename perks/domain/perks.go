package domain

type RegisterInterest struct {
	UserEmail string `json:"userEmail" binding:"required"`
	PerkName  string `json:"perkName" binding:"required"`
	ClickedOn string `json:"clickedOn" binding:"required"`
}
