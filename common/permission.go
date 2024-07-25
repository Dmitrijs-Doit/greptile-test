package common

type Permission string

// Available user permissions
const (
	PermissionBillingProfiles     Permission = "1SmYWoSAO1frHKjt34Gz"
	PermissionCloudAnalytics      Permission = "sfmBZeLN8uXWooCqJ4NO"
	PermissionFlexibleRI          Permission = "tvQnB14mSGr8LSU8reYH"
	PermissionInvoices            Permission = "HN6A3cPzDBcAIlc3ncDy"
	PermissionAssetsManager       Permission = "wfDH3k1FmYKHlQBwGIzZ"
	PermissionSettings            Permission = "AIzQjXTUQDgeZjXqNsgF"
	PermissionSandboxAdmin        Permission = "jnrMNJLdzRsyLCHCfq6T"
	PermissionSandboxUser         Permission = "KpogRUOHgMlroIH8xOUQ"
	PermissionIAM                 Permission = "ZqLGIVDUhNiSEDtrEb0S"
	PermissionContractsViewer     Permission = "8zXuFyohNSiiLy2ZQ6Xu"
	PermissionAnomaliesViewer     Permission = "eUKNGKekajR0NbOkHFfC"
	PermissionPerksViewer         Permission = "itIlDPCy18ymtEVgaW0B"
	PermissionIssuesViewer        Permission = "dEJbIiUcHn8GhW7IiWLW"
	PermissionBudgetsManager      Permission = "BgYDGr8dABLKjUkys7AD"
	PermissionMetricsManager      Permission = "fSFpOG5xUeHPlYVI5N1k"
	PermissionAttributionsManager Permission = "AnJW2Hwipmucak00yko0"
	PermissionSupportRequester    Permission = "jg1YHuQhsRlg5msNhpZZ"
	PermissionCAOwnerRoleAssigner Permission = "pwWRo04l9uXUYa8rIQSW"
	PermissionUsersManager        Permission = "ZqLGIVDUhNiSEDtrEb0S"
	PermissionLabelsManager       Permission = "CLxHZjNiVpjK83TA4h8S"
	PermissionDataHubAdmin        Permission = "Ss3OzvfCQQVFiRjUZQSp"
)

type PresetRole string

// Available preset user roles
const (
	PresetRoleAdmin        PresetRole = "59w2TJPTCa3XPsJ3KITY"
	PresetRolePowerUser    PresetRole = "Y2IN1X2rmWoZhTJDsYAN"
	PresetRoleITManager    PresetRole = "K634tvqvUvQGLPnqtubh"
	PresetRoleFinanceUser  PresetRole = "vkrmC1ioimecNyJ6vjPZ"
	PresetRoleStandardUser PresetRole = "e66fNltAJjdtpHwaXb6I"
	PresetRoleSupportUser  PresetRole = "YcZKjDZJHHUAl9yPCBmy"
)
