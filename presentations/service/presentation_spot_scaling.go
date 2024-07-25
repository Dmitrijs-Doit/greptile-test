package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/spot0/api/model"
)

const (
	sourceCustomerID = "lZJz236UZQ4coAS6mYLT"
	subnetPrefix     = "subnet-"
)

var anonymizedLaunchTemplate = map[string]string{
	"LaunchTemplateId":   "lt-0ad396d1ae8660d3f",
	"LaunchTemplateName": "lt-0ad396d1ae8660d3f",
	"Version":            "1",
}

func (p *PresentationService) CopySpotScalingDataToCustomer(ctx *gin.Context, customerID string) error {
	customer, err := p.getDemoCustomerFromID(ctx, customerID)
	if err != nil {
		return err
	}

	if err = p.doCopySpotScalingDataToCustomer(ctx, customer); err != nil {
		return err
	}

	return nil
}

func (p *PresentationService) CopySpotScalingDataToCustomers(ctx *gin.Context) error {
	docSnaps, err := p.customersDAL.GetPresentationCustomersWithAssetType(ctx, common.Assets.AmazonWebServices)
	if err != nil {
		return fmt.Errorf(FetchCustomerErr, err)
	}

	for _, docSnap := range docSnaps {
		var customer common.Customer

		if err := docSnap.DataTo(&customer); err != nil {
			return err
		}

		customer.Snapshot = docSnap
		customer.ID = docSnap.Ref.ID

		if err = p.doCopySpotScalingDataToCustomer(ctx, &customer); err != nil {
			return err
		}
	}

	return nil
}

func (p *PresentationService) doCopySpotScalingDataToCustomer(ctx context.Context, customer *common.Customer) error {
	cloudConnectData, err := createAWSCloudConnectForCustomer(ctx, customer)
	if err != nil {
		return err
	}

	fs := p.conn.Firestore(ctx)

	sourceCustomerRef := fs.Collection("customers").Doc(sourceCustomerID)

	asgsSnaps, err := fs.Collection("spot0").Doc("spotApp").Collection("asgs").
		Where("customer", "==", sourceCustomerRef).
		Where("error", "==", "").
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, asgSnap := range asgsSnaps {
		var asgConf model.AsgConfiguration
		if err := asgSnap.DataTo(&asgConf); err != nil {
			return err
		}

		if err := anonymizeAsgConf(customer, cloudConnectData.AccountID, &asgConf); err != nil {
			return err
		}

		asgConfID := fmt.Sprintf("%s_%s_%s", customer.ID, asgConf.Region, asgConf.ExecID)

		_, err = fs.Collection("spot0").Doc("spotApp").Collection("asgs").Doc(asgConfID).Set(ctx, asgConf)
		if err != nil {
			return err
		}
	}

	return nil
}

func createAWSCloudConnectForCustomer(
	ctx context.Context,
	customer *common.Customer,
) (*cloudconnect.AmazonWebServicesCredential, error) {
	cloudConnectData := cloudconnect.AmazonWebServicesCredential{
		AccountID:     awsDemoBillingAccountID,
		Customer:      customer.Snapshot.Ref,
		Status:        common.CloudConnectStatusTypeHealthy,
		CloudPlatform: common.Assets.AmazonWebServices,
		RoleID:        domain.CloudConnectRole,
		RoleName:      domain.CloudConnectRole,
		SupportedFeatures: []cloudconnect.SupportedFeature{
			{
				Name:                   "spot-scaling",
				HasRequiredPermissions: true,
			},
		},
	}

	if _, err := customer.Snapshot.Ref.
		Collection(domain.CloudConnectCollection).
		Doc(fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, cloudConnectData.AccountID)).
		Set(ctx, cloudConnectData); err != nil {
		return nil, errors.New("failed to create cloud connect data with error: " + err.Error())
	}

	return &cloudConnectData, nil
}

func anonymizeAsgConf(customer *common.Customer, accountID string, asgConf *model.AsgConfiguration) error {
	hexLetters := domain.GetCustomerHexLetters(customer.ID)
	asgID := fmt.Sprint(domain.Hash(customer.ID + asgConf.AsgName))

	asgConf.AccountID = accountID
	asgConf.AccountName = "doit-intl"
	asgConf.AsgName = anonymizeAsgName(customer.ID, asgConf.AsgName)
	asgConf.Customer = customer.Snapshot.Ref
	asgConf.ExecID = asgID
	asgConf.CloudFormationStack = ""
	asgConf.Spotisize.CurExcludedSubnets = nil

	if err := anonymizeSubnetDetails(hexLetters, asgConf.SubnetsDetails.(map[string]interface{})); err != nil {
		return err
	}

	if err := anonymizeSubnetList(hexLetters, asgConf.Config.ExcludedSubnets); err != nil {
		return err
	}

	if err := anonymizeAsg(hexLetters, asgID, asgConf, &asgConf.Spotisize.CurAsg); err != nil {
		return err
	}

	if err := anonymizeAsg(hexLetters, asgID, asgConf, &asgConf.Spotisize.RecAsg); err != nil {
		return err
	}

	return nil
}

func anonymizeAsg(hexLetters []string, asgID string, asgConf *model.AsgConfiguration, asg *model.Asg) error {
	asg.AutoScalingGroupName = asgConf.AsgName
	asg.AutoScalingGroupARN = fmt.Sprintf("arn:aws:autoscaling:%s:%s:autoScalingGroup:%s:autoScalingGroupName/%s", asgConf.Region, asgConf.AccountID, asgID, asgConf.AsgName)
	asg.ServiceLinkedRoleARN = fmt.Sprintf("arn:aws:iam::%s:role/aws-service-role/autoscaling.amazonaws.com/AWSServiceRoleForAutoScaling", asgConf.AccountID)
	asg.LaunchConfigurationName = nil

	if asg.LaunchTemplate != nil {
		asg.LaunchTemplate = anonymizedLaunchTemplate
	}

	asg.LoadBalancerNames = nil
	asg.Tags = nil
	asg.TargetGroupARNs = []string{asgConf.Spotisize.CurAsg.AutoScalingGroupARN}
	asg.TerminationPolicies = []string{"Default"}

	vpcZoneIdentifier, err := anonymizeVpcZoneIdentifier(hexLetters, asg.VPCZoneIdentifier)
	if err != nil {
		return err
	}

	asg.VPCZoneIdentifier = vpcZoneIdentifier

	return nil
}

func anonymizeAsgName(customerID string, originalName string) string {
	numericID := int(domain.Hash(customerID + originalName))

	return strings.Join([]string{
		projectPrefixes[numericID%len(projectPrefixes)],
		projectSuffixes[numericID%len(projectSuffixes)],
	}, "-")
}

func anonymizeHexIdentifier(hexLetters []string, prefix string, hexIdentifier string) (string, error) {
	if hexIdentifier == "" {
		return hexIdentifier, nil
	}

	output := prefix

	for _, letter := range strings.TrimPrefix(hexIdentifier, prefix) {
		index, err := strconv.ParseInt(string(letter), 16, 64)
		if err != nil {
			return "", err
		}

		output += hexLetters[index]
	}

	return output, nil
}

func anonymizeSubnetList(hexLetters []string, subnets []string) error {
	if subnets == nil {
		return nil
	}

	for idx, subnet := range subnets {
		anonymizedSubnet, err := anonymizeHexIdentifier(hexLetters, subnetPrefix, subnet)
		if err != nil {
			return err
		}

		subnets[idx] = anonymizedSubnet
	}

	return nil
}

func anonymizeSubnetDetails(hexLetters []string, subnetDetails map[string]interface{}) error {
	if subnetDetails == nil {
		return nil
	}

	for subnet, details := range subnetDetails {
		anonymizedSubnet, err := anonymizeHexIdentifier(hexLetters, subnetPrefix, subnet)
		if err != nil {
			return err
		}

		subnetDetails[anonymizedSubnet] = details
		delete(subnetDetails, subnet)
	}

	return nil
}

func anonymizeVpcZoneIdentifier(hexLetters []string, vpcZoneIdentifier string) (string, error) {
	if vpcZoneIdentifier == "" {
		return vpcZoneIdentifier, nil
	}

	const separator = ","

	anonymizedVpcZoneIdentifier := strings.Split(vpcZoneIdentifier, separator)

	if err := anonymizeSubnetList(hexLetters, anonymizedVpcZoneIdentifier); err != nil {
		return "", err
	}

	return strings.Join(anonymizedVpcZoneIdentifier, separator), nil
}
