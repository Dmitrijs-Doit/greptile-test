package domain

type DoitRole string

const (
	DoitRoleAwsAccountGenerator        DoitRole = "aws-account-generator-admin"
	DoitRoleBillingProfileAdmin        DoitRole = "billing-profile-admin"
	DoitRoleCustomerSettingsAdmin      DoitRole = "customer-settings-admin"
	DoitRoleDevelopers                 DoitRole = "developers"
	DoitRoleFlexsaveAdmin              DoitRole = "flexsave-admin"
	DoitRoleFlexsaveSuperAdmin         DoitRole = "flexsave-super-admin"
	DoitRoleMarketplaceAdmin           DoitRole = "marketplace-admin"
	DoitRoleMasterPayerAccountOpsAdmin DoitRole = "master-payer-account-ops-admin"
	DoitRoleOwners                     DoitRole = "owners"
	DoitRolePerksAdmin                 DoitRole = "perks-admin"
	DoitRoleContractAdmin              DoitRole = "contract-admin"
	DoitRoleContractOwner              DoitRole = "contract-owner"
	DoitRoleFieldSalesManager          DoitRole = "field-sales-manager"
	DoitRoleNordiaPreview              DoitRole = "nordia-feature-preview"
	DoitRoleCAOwnershipAssigner        DoitRole = "ca-ownership-assigner"
	DoitRoleTeamBruteforce             DoitRole = "team-bruteforce"
	DoitRoleCloudflow                  DoitRole = "cloudflow"
	DoitRoleCATemplateLibraryAdmin     DoitRole = "cloud-analytics-template-library-admins"
	DoitRoleCustomerTieringAdmin       DoitRole = "customer-tiering-admin"
)

func (r DoitRole) String() string {
	return string(r)
}
