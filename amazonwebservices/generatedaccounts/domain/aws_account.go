package domain

type AwsAccountCompletionStep string

const (
	AwsAccountCompletionStepCredentials          = "CREDENTIALS"
	AwsAccountCompletionStepAccountData          = "ACCOUNT_DATA"
	AwsAccountCompletionStepPaymentInformation   = "PAYMENT_INFORMATION"
	AwsAccountCompletionStepIdentityConfirmation = "IDENTITY_CONFIRMATION"
	AwsAccountCompletionStepAdminRoles           = "ADMIN_ROLES"
	AwsAccountCompletionStepGenuineAwsAccountID  = "GENUINE_AWS_ACCOUNT_ID"
	AwsAccountCompletionStepAccountAlias         = "ACCOUNT_ALIAS"
)

type AwsAccount struct {
	AccountName         string                     `firestore:"accountName"`
	Email               string                     `firestore:"email"`
	CompleteSteps       []AwsAccountCompletionStep `firestore:"completeSteps"`
	GenuineAwsAccountID *string                    `firestore:"genuineAwsAccountId"`
}
