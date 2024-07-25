package microsoft

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

const (
	StatusActive    string = "active"
	StatusSuspended string = "suspended"
	StatusPending   string = "pending"
)

type SecureApplicationModelConfig struct {
	Resource     string `json:"resource"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Domain       string `json:"domain"`
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
}

const (
	AzureResellerMarginModifier float64 = 0.85
)

// All CSP names
const (
	CSPDomainIL     CSPDomain = "doitintl.onmicrosoft.com"
	CSPDomainUS     CSPDomain = "doitintlus.onmicrosoft.com"
	CSPDomainEU     CSPDomain = "doitintleu.onmicrosoft.com"
	CSPDomainEurope CSPDomain = "doitintleurope.onmicrosoft.com"
	CSPDomainUK     CSPDomain = "doitintluk.onmicrosoft.com"
	CSPDomainAU     CSPDomain = "doitintlau.onmicrosoft.com"
)

type AzureBillingAccount string

// All CSPs Azure billing accounts
const (
	AzureBillingAccountIL     AzureBillingAccount = "2be39f01-04d5-4872-a8ce-2d651d4e140a:ab9ee279-a6f0-46f7-9db2-3fc6369dcf03_2018-09-30"
	AzureBillingAccountUS     AzureBillingAccount = "142959a3-06e1-5217-6bc5-ef48bf34745d:0f686be0-3804-4d7e-8970-0204b4968393_2019-05-31"
	AzureBillingAccountEU     AzureBillingAccount = "a29f28b8-d564-52e4-c6fa-3af49e9b7bca:659227ae-cc9f-4f33-8bcc-e9ffa5a32433_2019-05-31"
	AzureBillingAccountUK     AzureBillingAccount = "28ab8e44-fc77-5fc1-3c42-674a63add3f2:4e381586-f9b8-4d2a-8641-cc57240ce8d0_2019-05-31"
	AzureBillingAccountEurope AzureBillingAccount = "77c66599-648d-57ee-dadf-230c5b7a6471:bf0dfe00-73a1-4468-9dc2-d54f43a3a781_2019-05-31"
	AzureBillingAccountAU     AzureBillingAccount = "2de74db4-4269-5799-d07e-3427d3435626:a65683bb-7e77-4684-a5d5-de916469141e_2019-05-31"
)

var AzureBillingAccounts = map[CSPDomain]AzureBillingAccount{
	CSPDomainIL:     AzureBillingAccountIL,
	CSPDomainUS:     AzureBillingAccountUS,
	CSPDomainEU:     AzureBillingAccountEU,
	CSPDomainUK:     AzureBillingAccountUK,
	CSPDomainEurope: AzureBillingAccountEurope,
	CSPDomainAU:     AzureBillingAccountAU,
}

// Microsoft partner center and Azure service management access tokens
var (
	MPCAccessTokens []*AccessToken
	ASMAccessTokens []*AccessToken
)

func init() {
	ctx := context.Background()

	// Get Microsoft Partner Center refresh token and secret
	var mpcConfig []*SecureApplicationModelConfig

	mpcData, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretMPC)
	if err != nil {
		log.Fatalln(err)
	}

	if err = json.Unmarshal(mpcData, &mpcConfig); err != nil {
		log.Fatalln(err)
	}

	for _, conf := range mpcConfig {
		var accessToken AccessToken
		accessToken.mutex = &sync.Mutex{}
		accessToken.config = conf
		accessToken.secret = secretmanager.SecretMPC
		accessToken.Resource = conf.Resource
		MPCAccessTokens = append(MPCAccessTokens, &accessToken)
	}

	// Get Azure Service Management refresh token and secret
	var asmConfig []*SecureApplicationModelConfig

	asmData, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretASM)
	if err != nil {
		log.Fatalln(err)
	}

	if err = json.Unmarshal(asmData, &asmConfig); err != nil {
		log.Fatalln(err)
	}

	for _, conf := range asmConfig {
		var accessToken AccessToken
		accessToken.mutex = &sync.Mutex{}
		accessToken.config = conf
		accessToken.secret = secretmanager.SecretASM
		ASMAccessTokens = append(ASMAccessTokens, &accessToken)
	}

	go updateRefreshTokens(ctx, secretmanager.SecretMPC, MPCAccessTokens)
	go updateRefreshTokens(ctx, secretmanager.SecretASM, ASMAccessTokens)
}

// updateRefreshTokens updates microsoft refresh tokens (valid up to 90 days)
func updateRefreshTokens(ctx context.Context, secret secretmanager.SecretName, accessTokens []*AccessToken) error {
	secretVersion, err := secretmanager.GetSecretLatestVersion(ctx, secret)
	if err != nil {
		return err
	}

	// Refresh token will expire after 90 days, save new refresh token every 30 days
	expiryDate := secretVersion.GetCreateTime().AsTime().UTC().AddDate(0, 0, 30)
	if time.Now().UTC().After(expiryDate) {
		newSecret := make([]*SecureApplicationModelConfig, len(accessTokens))

		for i, accessToken := range accessTokens {
			if err := accessToken.Refresh(); err != nil {
				return err
			}

			newSecret[i] = &SecureApplicationModelConfig{
				Resource:     accessToken.config.Resource,
				ClientID:     accessToken.config.ClientID,
				ClientSecret: accessToken.config.ClientSecret,
				Domain:       accessToken.config.Domain,
				GrantType:    accessToken.config.GrantType,
				RefreshToken: accessToken.RefreshToken,
			}
		}

		data, err := json.MarshalIndent(newSecret, "", "    ")
		if err != nil {
			return err
		}

		if _, err := secretmanager.AddSecretVersion(ctx, secret, data); err != nil {
			return err
		}
	}

	return nil
}

// strips BOM from microsoft api responses
func stripBOM(bs []byte) []byte {
	return bytes.TrimPrefix(bs, []byte("\uFEFF"))
}

func getCustomerRefByDomain(ctx *gin.Context, fs *firestore.Client, domain string) (*firestore.DocumentRef, error) {
	customerRef := fb.Orphan

	docSnaps, err := fs.Collection("customers").Where("domains", "array-contains", domain).Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	if len(docSnaps) > 0 {
		customerRef = docSnaps[0].Ref
	}

	return customerRef, nil
}
