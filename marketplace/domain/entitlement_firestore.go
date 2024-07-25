package domain

import "time"

type EntitlementState string

const (
	EntitlementStateStateUnspecified          EntitlementState = "ENTITLEMENT_STATE_UNSPECIFIED"
	EntitlementStateActivationRequested       EntitlementState = "ENTITLEMENT_ACTIVATION_REQUESTED"
	EntitlementStateActive                    EntitlementState = "ENTITLEMENT_ACTIVE"
	EntitlementStatePendingCancellation       EntitlementState = "ENTITLEMENT_PENDING_CANCELLATION"
	EntitlementStateCancelled                 EntitlementState = "ENTITLEMENT_CANCELLED"
	EntitlementStatePendingPlanChange         EntitlementState = "ENTITLEMENT_PENDING_PLAN_CHANGE"
	EntitlementStatePendingPlanChangeApproval EntitlementState = "ENTITLEMENT_PENDING_PLAN_CHANGE_APPROVAL"
	EntitlementStateSuspended                 EntitlementState = "ENTITLEMENT_SUSPENDED"
	EntitlementStateDeleted                   EntitlementState = "ENTITLEMENT_DELETED"
)

type EntitlementFirestore struct {
	ProcurementEntitlement *ProcurementEntitlementFirestore `json:"procurementEntitlement,omitempty" firestore:"procurementEntitlement,omitempty"`
}

type ProcurementEntitlementFirestore struct {
	Account                 string           `json:"account,omitempty" firestore:"account,omitempty"`
	CreateTime              time.Time        `json:"createTime,omitempty" firestore:"createTime,omitempty"`
	MessageToUser           string           `json:"messageToUser,omitempty" firestore:"messageToUser,omitempty"`
	Name                    string           `json:"name,omitempty" firestore:"name,omitempty"`
	NewPendingOffer         string           `json:"newPendingOffer,omitempty" firestore:"newPendingOffer,omitempty"`
	NewPendingOfferDuration string           `json:"newPendingOfferDuration,omitempty" firestore:"newPendingOfferDuration,omitempty"`
	NewPendingPlan          string           `json:"newPendingPlan,omitempty" firestore:"newPendingPlan,omitempty"`
	Offer                   string           `json:"offer,omitempty" firestore:"offer,omitempty"`
	OfferDuration           string           `json:"offerDuration,omitempty" firestore:"offerDuration,omitempty"`
	OfferEndTime            string           `json:"offerEndTime,omitempty" firestore:"offerEndTime,omitempty"`
	Plan                    string           `json:"plan,omitempty" firestore:"plan,omitempty"`
	Product                 string           `json:"product,omitempty" firestore:"product,omitempty"`
	ProductExternalName     string           `json:"productExternalName,omitempty" firestore:"productExternalName,omitempty"`
	Provider                string           `json:"provider,omitempty" firestore:"provider,omitempty"`
	QuoteExternalName       string           `json:"quoteExternalName,omitempty" firestore:"quoteExternalName,omitempty"`
	State                   EntitlementState `json:"state,omitempty" firestore:"state,omitempty"`
	SubscriptionEndTime     string           `json:"subscriptionEndTime,omitempty" firestore:"subscriptionEndTime,omitempty"`
	UpdateTime              time.Time        `json:"updateTime,omitempty" firestore:"updateTime,omitempty"`
	UsageReportingID        string           `json:"usageReportingId,omitempty" firestore:"usageReportingId,omitempty"`
}
