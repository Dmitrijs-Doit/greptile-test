package domain

type StripeAccountID string

const (
	StripeAccountUS     StripeAccountID = "acct_1A5pxKKD42kkxF15"
	StripeAccountUKandI StripeAccountID = "acct_1HTM1zLxqYtCDmwM"
	StripeAccountDE     StripeAccountID = "acct_1Myzw1JCzGwjua24"
)

var StripeAccountNames = map[StripeAccountID]string{
	StripeAccountUS:     "DoiT International (US)",
	StripeAccountDE:     "DoiT International, DACH GmbH",
	StripeAccountUKandI: "DoiT International UK&I",
}
