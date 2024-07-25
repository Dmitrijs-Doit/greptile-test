package amazonwebservices

import (
	"context"
	"reflect"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/aws/aws-sdk-go/service/support"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/quotas"
	notificationcenterDomain "github.com/doitintl/notificationcenter/domain"
	notificationcenterClient "github.com/doitintl/notificationcenter/pkg"
	notificationcenter "github.com/doitintl/notificationcenter/service"
)

type ServiceLimits struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
}

func NewServiceLimits(loggerProvider logger.Provider, conn *connection.Connection) *ServiceLimits {
	return &ServiceLimits{
		loggerProvider,
		conn,
	}
}

type CheckResult struct {
	Limit              string `firestore:"limit"`
	Usage              string `firestore:"usage"`
	Status             string `firestore:"status"`
	Region             string `firestore:"region"`
	RefreshTime        string `firestore:"refreshTime"`
	ServiceDescription string `firestore:"serviceDescription"`
	CheckID            string `firestore:"checkId"`
	IsEmailSent        bool   `firestore:"isEmailSent"`
	AccountID          string `firestore:"accountId"`
}

var serviceLimitsRequiredPermission = cloudconnect.SupportedFeature{
	Name:                   "quotas",
	HasRequiredPermissions: true,
}

func (s *ServiceLimits) GetCustomerServicesLimits(ctx context.Context) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	docSnaps, err := fs.CollectionGroup("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.AmazonWebServices).
		Where("supportedFeatures", common.ArrayContains, serviceLimitsRequiredPermission).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range docSnaps {
		var cloudConnectCred cloudconnect.AmazonWebServicesCredential
		if err := docSnap.DataTo(&cloudConnectCred); err != nil {
			l.Errorf("DataTo failed with error: %s", err)
			continue
		}

		if err := s.RefreshChecks(ctx, cloudConnectCred); err != nil {
			l.Errorf("RefreshChecks for account %s failed with error: %s", cloudConnectCred.AccountID, err)
			continue
		}
	}

	return nil
}

func (s *ServiceLimits) UpdateCustomerLimit(ctx context.Context, customerID string) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	docSnaps, err := fs.Collection("customers").Doc(customerID).Collection("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.AmazonWebServices).
		Where("supportedFeatures", common.ArrayContains, serviceLimitsRequiredPermission).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range docSnaps {
		var cloudConnectCred cloudconnect.AmazonWebServicesCredential
		if err := docSnap.DataTo(&cloudConnectCred); err != nil {
			l.Errorf("DataTo failed with error: %s", err)
			continue
		}

		if err := s.RefreshChecks(ctx, cloudConnectCred); err != nil {
			l.Errorf("RefreshChecks for account %s failed with error: %s", cloudConnectCred.AccountID, err)
		}
	}

	return nil
}

func (s *ServiceLimits) getServiceLimitsForCred(
	ctx context.Context,
	cloudConnectCred cloudconnect.AmazonWebServicesCredential,
	sess *session.Session,
	creds *credentials.Credentials,
) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	supportService := support.New(sess, aws.NewConfig().WithCredentials(creds))
	input := &support.DescribeTrustedAdvisorChecksInput{
		Language: aws.String("en"),
	}

	output, err := supportService.DescribeTrustedAdvisorChecks(input)
	if err != nil {
		return err
	}

	stsService := sts.New(sess, aws.NewConfig().WithCredentials(creds))
	inputIam := &sts.GetCallerIdentityInput{}

	accountResult, err := stsService.GetCallerIdentity(inputIam)
	if err != nil {
		return err
	}

	customerRes, err := fs.Collection("integrations").Doc("amazon-web-services").Collection("service-limits").Doc(cloudConnectCred.Customer.ID).Get(ctx)
	if err != nil {
		return err
	}

	var allLimitsForEmail []quotas.EmailLimit

	awsAccounts := make(map[string]interface{})
	serviceLimits := make(map[string]map[string][]CheckResult)

	for _, check := range output.Checks {
		inputCheck := &support.DescribeTrustedAdvisorCheckResultInput{
			CheckId: check.Id,
		}

		res, err := supportService.DescribeTrustedAdvisorCheckResult(inputCheck)
		if err != nil {
			continue
		}

		refreshTime := aws.StringValue(res.Result.Timestamp)
		serviceDescription := aws.StringValue(check.Description)

		for _, resource := range res.Result.FlaggedResources {
			m := aws.StringValueSlice(resource.Metadata)

			if len(m) > 5 && aws.StringValue(resource.Status) != "error" && aws.StringValue(check.Category) == "service_limits" {
				region := m[0]
				serviceName := m[1]
				serviceNameDetails := m[2]
				serviceLimit := m[3]
				serviceUsage := m[4]
				serviceStatus := m[5]
				usageInt, _ := strconv.Atoi(serviceUsage)
				limitInt, _ := strconv.Atoi(serviceLimit)
				addToFirestore := false

				var currentUsagePercent int
				if limitInt > 0 {
					currentUsagePercent = (usageInt * 100) / limitInt
					if currentUsagePercent > 50 {
						addToFirestore = true
					}
				}

				if serviceName != "Not enabled" && serviceName != "Disabled" && serviceUsage != "0" && serviceUsage != "" && addToFirestore {
					if serviceLimits[serviceName] == nil {
						serviceLimits[serviceName] = make(map[string][]CheckResult)
					}

					isSendEmail := false

					if serviceStatus == "Yellow" || serviceStatus == "Red" {
						// Send email
						arrMap, err := customerRes.DataAtPath([]string{"services", cloudConnectCred.AccountID, serviceName, serviceNameDetails})
						if err == nil {
							t := arrMap
							switch reflect.TypeOf(t).Kind() {
							case reflect.Slice:
								s := reflect.ValueOf(t)
								for i := 0; i < s.Len(); i++ {
									if s.Index(i).Interface().(map[string]interface{})["region"] == region {
										if s.Index(i).Interface().(map[string]interface{})["isEmailSent"] == nil {
											isSendEmail = false
										} else {
											isSendEmail = s.Index(i).Interface().(map[string]interface{})["isEmailSent"].(bool)
										}
									}
								}
							}
						}

						var limitObj quotas.EmailLimit
						limitObj.Limit = strconv.Itoa(currentUsagePercent) + "%"
						limitObj.Region = region
						limitObj.Service = serviceName + " - " + serviceNameDetails
						limitObj.AccountID = aws.StringValue(accountResult.Account)

						statusName := "Limit Reached"
						if serviceStatus == "Yellow" {
							statusName = "Warning"
						}

						limitObj.Status = statusName

						if !isSendEmail {
							isSendEmail = true

							allLimitsForEmail = append(allLimitsForEmail, limitObj)
						}
					}

					serviceLimits[serviceName][serviceNameDetails] = append(serviceLimits[serviceName][serviceNameDetails], CheckResult{
						Limit:              serviceLimit,
						Usage:              serviceUsage,
						Status:             serviceStatus,
						Region:             region,
						RefreshTime:        refreshTime,
						ServiceDescription: serviceDescription,
						CheckID:            aws.StringValue(check.Id),
						IsEmailSent:        isSendEmail,
						AccountID:          aws.StringValue(accountResult.Account),
					})
					awsAccounts[cloudConnectCred.AccountID] = serviceLimits
				}
			}
		}
	}

	if len(allLimitsForEmail) > 0 {
		customerData, err := fs.Collection("customers").Doc(cloudConnectCred.Customer.ID).Get(ctx)
		if err != nil {
			return err
		}

		notificationService := notificationcenter.NewRecipientsService(fs)

		notificationClient, err := notificationcenterClient.NewClient(ctx, common.ProjectID)
		if err != nil {
			l.Error("Failed to create notification client")
			return err
		}

		recipients, err := notificationService.GetNotificationRecipientsForCustomer(ctx, cloudConnectCred.Customer, notificationcenterDomain.NotificationCloudQuotaUtilization)
		if err != nil {
			l.Error("Failed to get notification recipients")
			return err
		}

		emailTo, emailCc, slackChannels := quotas.GetQuotaNotificationTargets(ctx, recipients, cloudConnectCred.Customer.ID, common.AccountManagerCompanyAws, fs)

		primaryDomain, ok := customerData.Data()["primaryDomain"].(string)
		if !ok {
			l.Info("primaryDomain of customer %s is not a string", cloudConnectCred.Customer.ID)

			primaryDomain = ""
		}

		notificationToSend := quotas.CreateQuotaNotification(quotas.QuotaNotificationData{
			EmailTo:        emailTo,
			EmailCc:        emailCc,
			PrimaryDomain:  primaryDomain,
			Limits:         allLimitsForEmail,
			Platform:       "AWS",
			Link:           "https://aws.amazon.com/support/createCase?type=service_limit_increase",
			Documentation:  "https://docs.aws.amazon.com/general/latest/gr/aws_service_limits.html",
			SlackChannells: slackChannels,
		})

		task, err := notificationClient.CreateSendTask(ctx, notificationToSend)
		if err != nil {
			l.Errorf("Failed to send AWS cloud quota utilization notification for customer: %s, with task name:%s and error: %v", cloudConnectCred.Customer.ID, task.GetName(), err)
			return err
		} else {
			l.Infof("Sent AWS cloud quota utilization notification for customer: %s, with task name: %s", cloudConnectCred.Customer.ID, task.GetName())
		}
	}

	if _, err := fs.Collection("integrations").Doc("amazon-web-services").Collection("service-limits").Doc(cloudConnectCred.Customer.ID).Set(ctx, map[string]interface{}{
		"services": awsAccounts,
		"customer": cloudConnectCred.Customer,
	}, firestore.MergeAll); err != nil {
		return err
	}

	return nil
}

func (s *ServiceLimits) RefreshChecks(ctx context.Context, cloudConnectCred cloudconnect.AmazonWebServicesCredential) error {
	l := s.loggerProvider(ctx)

	awscreds, err := cloudconnect.GetAWSCredentials()
	if err != nil {
		return err
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(endpoints.UsEast1RegionID),
		Credentials: awscreds,
	})
	if err != nil {
		return err
	}

	creds := stscreds.NewCredentials(sess, cloudConnectCred.Arn, func(arp *stscreds.AssumeRoleProvider) {
		arp.RoleSessionName = "doit-service-limits"
		arp.Duration = 60 * time.Minute
		arp.ExternalID = aws.String(cloudConnectCred.Customer.ID)
	})

	supportService := support.New(sess, aws.NewConfig().WithCredentials(creds))
	input := &support.DescribeTrustedAdvisorChecksInput{
		Language: aws.String("en"),
	}

	output, err := supportService.DescribeTrustedAdvisorChecks(input)
	if err != nil {
		return err
	}

	for _, check := range output.Checks {
		input := &support.RefreshTrustedAdvisorCheckInput{
			CheckId: check.Id,
		}

		if _, err := supportService.RefreshTrustedAdvisorCheck(input); err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "InvalidParameterValueException" {
					// "UNREFRESHABLE_CHECK_ID_ERROR" - The checkId provided is not refreshable
					// This spam the logs, so we ignore it
					continue
				}
			}

			l.Errorf("RefreshTrustedAdvisorCheck '%s' for account %s failed with error: %s", *input.CheckId, cloudConnectCred.AccountID, err)
		}
	}

	return s.getServiceLimitsForCred(ctx, cloudConnectCred, sess, creds)
}
