package servicecatalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	awsSdkServiceCatalog "github.com/aws/aws-sdk-go/service/servicecatalog"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/awsproxy"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const quotaLimit = 5000

type Portfolio struct {
	ID          string
	DisplayName string
}

type Client interface {
	CreatePortfolioShare(portfolioID string, accountID string) error
	DeletePortfolioShare(portfolioID string, accountID string) error
	GetPortfoliosByNamePrefix(prefix string) ([]Portfolio, error)
	IsShareQuotaReached(portfolioID string) (bool, error)
	GetAllSharesByNamePrefix(prefix string) (map[string]string, error)
}

type ServiceCatalogClient struct {
	Client *awsSdkServiceCatalog.ServiceCatalog
}

func (c *ServiceCatalogClient) CreatePortfolioShare(porfolioID string, accountID string) error {
	if !common.Production {
		return nil
	}

	input := &awsSdkServiceCatalog.CreatePortfolioShareInput{
		PortfolioId:     aws.String(porfolioID),
		AccountId:       aws.String(accountID),
		ShareTagOptions: aws.Bool(true),
	}

	_, err := c.Client.CreatePortfolioShare(input)

	return err
}

func (c *ServiceCatalogClient) DeletePortfolioShare(porfolioID string, accountID string) error {
	if !common.Production {
		return nil
	}

	input := &awsSdkServiceCatalog.DeletePortfolioShareInput{
		PortfolioId: aws.String(porfolioID),
		AccountId:   aws.String(accountID),
	}

	_, err := c.Client.DeletePortfolioShare(input)

	return err
}

func (c *ServiceCatalogClient) GetPortfoliosByNamePrefix(prefix string) ([]Portfolio, error) {
	input := &awsSdkServiceCatalog.ListPortfoliosInput{}
	matches := []Portfolio{}

	for {
		result, err := c.Client.ListPortfolios(input)
		if err != nil {
			return matches, err
		}

		for _, portfolio := range result.PortfolioDetails {
			if strings.HasPrefix(*portfolio.DisplayName, prefix) && *portfolio.ProviderName == "DoiT" {
				matches = append(matches, Portfolio{
					ID:          *portfolio.Id,
					DisplayName: *portfolio.DisplayName,
				})
			}
		}

		if result.NextPageToken == nil {
			break
		}

		input.PageToken = result.NextPageToken
	}

	return matches, nil
}

func (c *ServiceCatalogClient) IsShareQuotaReached(portfolioID string) (bool, error) {
	input := &awsSdkServiceCatalog.DescribePortfolioSharesInput{
		PortfolioId: aws.String(portfolioID),
		Type:        aws.String("ACCOUNT"),
	}

	count := 0

	for {
		result, err := c.Client.DescribePortfolioShares(input)
		if err != nil {
			return false, err
		}

		count += len(result.PortfolioShareDetails)

		if result.NextPageToken == nil {
			break
		}

		input.PageToken = result.NextPageToken
	}

	return count >= quotaLimit, nil
}

func (c *ServiceCatalogClient) GetAllSharesByNamePrefix(prefix string) (map[string]string, error) {
	var sharedWithAccounts = map[string]string{} // accountID -> portfolioID

	portfolios, err := c.GetPortfoliosByNamePrefix(prefix)
	if err != nil {
		return sharedWithAccounts, err
	}

	for _, portfolio := range portfolios {
		input := &awsSdkServiceCatalog.DescribePortfolioSharesInput{
			PortfolioId: aws.String(portfolio.ID),
			Type:        aws.String("ACCOUNT"),
		}

		for {
			result, err := c.Client.DescribePortfolioShares(input)
			if err != nil {
				return sharedWithAccounts, err
			}

			for _, share := range result.PortfolioShareDetails {
				sharedWithAccounts[*share.PrincipalId] = portfolio.ID
			}

			if result.NextPageToken == nil {
				break
			}

			input.PageToken = result.NextPageToken
		}
	}

	return sharedWithAccounts, nil
}

func GetProxySession() (*session.Session, error) {
	proxyCreds, err := awsproxy.NewCredentials()
	if err != nil {
		return nil, err
	}

	proxySession, err := session.NewSession(&aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
		Credentials: credentials.NewStaticCredentials(
			*proxyCreds.AccessKeyId,
			*proxyCreds.SecretAccessKey,
			*proxyCreds.SessionToken,
		),
	})
	if err != nil {
		return nil, fmt.Errorf("GetProxySession: could not initialize aws proxy session. error %s", err)
	}

	return proxySession, nil
}

func NewClientWithSession(proxySession *session.Session, roleArn string, region string) (*ServiceCatalogClient, error) {
	assumeRoleSession, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
		Credentials: stscreds.NewCredentials(proxySession, roleArn, func(arp *stscreds.AssumeRoleProvider) {
			arp.Duration = 60 * time.Minute
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("GetClientSession: could not initialize aws assumeRole session. error %s", err)
	}

	awsClient, err := awsSdkServiceCatalog.New(assumeRoleSession), nil
	if err != nil {
		return nil, fmt.Errorf("GetClientSession: could not initialize aws service catalog client. error %s", err)
	}

	return &ServiceCatalogClient{
		Client: awsClient,
	}, nil
}
