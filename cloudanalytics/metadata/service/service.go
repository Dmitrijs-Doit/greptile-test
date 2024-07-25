package service

import (
	"context"
	"sort"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/customerapi"
	assetDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	attributionGroupsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal"
	attributionGroupsDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/datahubmetric"
	metadataDalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	gcpMetadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/gcp"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	awsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/aws"
	awsIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/aws/iface"
	bqlensService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/bqlens"
	bqlensIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/bqlens/iface"
	datahubService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/datahub"
	datahubIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/datahub/iface"
	gcpService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/gcp"
	gcpIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/gcp/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	azureService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/microsoftazure"
	azureIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/microsoftazure/iface"
	utils "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/utils"
	cloudConnectDal "github.com/doitintl/hello/scheduled-tasks/cloudconnect/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
	userDalIface "github.com/doitintl/hello/scheduled-tasks/user/dal/iface"
	tier "github.com/doitintl/tiers/service"
)

type MetadataService struct {
	logger.Provider
	*connection.Connection
	azureMetadata        azureIface.AzureMetadata
	bqlensMetadata       bqlensIface.BQLensMetadata
	datahubMetadata      datahubIface.DataHubMetadata
	gcpMetadata          gcpIface.GCPMetadata
	awsMetadata          awsIface.AWSMetadata
	dal                  metadataDalIface.Metadata
	customerDal          customerDal.Customers
	userDal              userDalIface.IUserFirestoreDAL
	attributionGroupsDAL attributionGroupsDalIface.AttributionGroups
}

func NewMetadataService(ctx context.Context, loggerProvider logger.Provider, conn *connection.Connection) *MetadataService {
	bkt := datahubService.GetDataHubEventsBucket(ctx, conn)
	datahubMetadataGCSDal := dal.NewDataHubMetadataGCS(bkt)

	metadataDal := dal.NewMetadataFirestoreWithClient(conn.Firestore)
	customerDal := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)
	assetDal := assetDal.NewAssetsFirestoreWithClient(conn.Firestore)
	userDal := userDal.NewUserFirestoreDALWithClient(conn.Firestore)
	attributionGroupDal := attributionGroupsDal.NewAttributionGroupsFirestoreWithClient(conn.Firestore)
	cloudConnectDal := cloudConnectDal.NewGcpConnectWithClient(conn.Firestore)
	datahubMetadataFirestoreDal := dal.NewDataHubMetadataFirestoreWithClient(conn.Firestore)
	datahubMetricFirestoreDal := datahubmetric.NewDataHubMetricFirestoreWithClient(conn.Firestore)

	datahubMetadata := datahubService.NewDataHubMetadataService(
		loggerProvider,
		datahubMetadataFirestoreDal,
		datahubMetricFirestoreDal,
		datahubMetadataGCSDal,
		customerDal,
	)

	azureMetadata := azureService.NewAzureMetadataService(loggerProvider, conn, metadataDal, assetDal, customerDal)
	bqlensMetadata := bqlensService.NewBQLensMetadataService(loggerProvider, conn, metadataDal, assetDal, customerDal, cloudConnectDal)
	gcpMetadata := gcpService.NewGCPMetadataService(loggerProvider, conn, metadataDal)
	tierService := tier.NewTiersService(conn.Firestore)
	awsMetadata := awsService.NewAWSMetadataService(loggerProvider, conn, customerDal, tierService)

	return &MetadataService{
		loggerProvider,
		conn,
		azureMetadata,
		bqlensMetadata,
		datahubMetadata,
		gcpMetadata,
		awsMetadata,
		metadataDal,
		customerDal,
		userDal,
		attributionGroupDal,
	}
}

func (s *MetadataService) getOrganizationRef(ctx context.Context, isDoitEmployee bool, userID string, customerID string) (*firestore.DocumentRef, error) {
	if isDoitEmployee {
		return s.dal.GetPresetOrgRef(ctx, metadata.DoitOrgID), nil
	}

	if userID == "" {
		return nil, metadata.ErrServiceExpectingUserID
	}

	user, err := s.userDal.Get(ctx, userID)
	if err != nil {
		return nil, err
	}

	if user.Organizations == nil || len(user.Organizations) == 0 {
		return s.dal.GetCustomerOrgRef(ctx, customerID, metadata.RootOrgID), nil
	}

	return user.Organizations[0], nil
}

func (s *MetadataService) ExternalAPIList(args iface.ExternalAPIListArgs) (iface.ExternalAPIListRes, error) {
	orgRef, err := s.getOrganizationRef(args.Ctx, args.IsDoitEmployee, args.UserID, args.CustomerID)
	if err != nil {
		return nil, err
	}

	customerRef := s.customerDal.GetRef(args.Ctx, args.CustomerID)

	attrGroupsList, err := s.AttributionGroupsMetadata(args.Ctx, args.CustomerID, args.UserEmail)
	if err != nil {
		return nil, err
	}

	// sorting the attribution groups list by key
	sort.Slice(attrGroupsList, func(i, j int) bool {
		return attrGroupsList[i].Key < attrGroupsList[j].Key
	})

	typesFilter := []metadata.MetadataFieldType{
		metadata.MetadataFieldTypeFixed,
		metadata.MetadataFieldTypeDatetime,
		metadata.MetadataFieldTypeAttribution,
		metadata.MetadataFieldTypeOptional,
	}

	if !args.OmitGkeTypes {
		typesFilter = append(typesFilter, metadata.MetadataFieldTypeGKE)
	}

	listsByType, err := s.dal.ListMap(metadataDalIface.ListArgs{
		Ctx:         args.Ctx,
		CustomerRef: customerRef,
		OrgRef:      orgRef,
		TypesFilter: typesFilter,
	})

	if err != nil {
		return nil, err
	}

	// adding the attribution groups to the lists
	for _, item := range attrGroupsList {
		listsByType[item.Type] = append(listsByType[item.Type], metadataDalIface.ListItem{
			Key:       item.Key,
			Type:      item.Type,
			Label:     item.Label,
			Timestamp: item.Timestamp,
		})
	}

	s.mapListItemOptionals(listsByType)

	if args.OmitGkeTypes {
		// removing all GKE metadata fields from the list
		listsByType[metadata.MetadataFieldTypeGKE] = nil
		listsByType[metadata.MetadataFieldTypeGKELabel] = nil
	}

	sortedList := s.dal.FlatAndSortListMap(listsByType)

	res := make(iface.ExternalAPIListRes, 0)

	minTimestamp := time.Now().AddDate(0, 0, -30)
	for _, item := range sortedList {
		if args.OmitByTimestamp && item.Timestamp.Before(minTimestamp) {
			// if timestamp not after 30 days ago, don't add to list
			continue
		}

		candidate := iface.ExternalAPIListItem{
			ID:    item.Key,
			Type:  item.Type,
			Label: item.Label,
		}
		if !s.isListItemUnique(res, candidate) {
			continue
		}

		res = append(res, candidate)
	}

	return res, nil
}

func (s *MetadataService) ExternalAPIListWithFilters(args iface.ExternalAPIListArgs, req *customerapi.Request) (*domain.DimensionsExternalAPIList, error) {
	res, err := s.ExternalAPIList(args)
	if err != nil {
		return nil, err
	}

	filteredRes := customerapi.FilterAPIList(utils.ToDimensionsList(res), req.Filters)

	sortedDimensions, err := customerapi.SortAPIList(filteredRes, req.SortBy, req.SortOrder)
	if err != nil {
		return nil, err
	}

	page, token, err := customerapi.GetEncodedAPIPage(req.MaxResults, req.NextPageToken, sortedDimensions)
	if err != nil {
		return nil, err
	}

	return &domain.DimensionsExternalAPIList{
		RowCount:   len(page),
		Dimensions: page,
		PageToken:  token,
	}, nil
}
func (s *MetadataService) ExternalAPIGet(args iface.ExternalAPIGetArgs) (*iface.ExternalAPIGetRes, error) {
	orgRef, err := s.getOrganizationRef(args.Ctx, args.IsDoitEmployee, args.UserID, args.CustomerID)
	if err != nil {
		return nil, err
	}

	customerRef := s.customerDal.GetRef(args.Ctx, args.CustomerID)
	combinedList := make([]metadataDalIface.GetItem, 0)

	attrGroupsList, err := s.AttributionGroupsMetadata(args.Ctx, args.CustomerID, args.UserEmail)
	if err != nil {
		return nil, err
	}

	filteredAttrGroupsList := s.filterAttrGroups(attrGroupsList, args.KeyFilter, args.TypeFilter)
	combinedList = append(combinedList, filteredAttrGroupsList...)

	metadataList, err := s.dal.Get(metadataDalIface.GetArgs{
		Ctx:         args.Ctx,
		CustomerRef: customerRef,
		OrgRef:      orgRef,
		KeyFilter:   args.KeyFilter,
		TypeFilter:  args.TypeFilter,
	})

	if err != nil {
		return nil, err
	}

	combinedList = append(combinedList, metadataList...)

	if len(combinedList) == 0 {
		return nil, metadata.ErrNotFound
	}

	res := iface.ExternalAPIGetRes{
		ID:    combinedList[0].Key,
		Type:  combinedList[0].Type,
		Label: combinedList[0].Label,
	}

	res.Values = make([]iface.ExternalAPIGetValue, 0)

	for _, item := range combinedList {
		// sorting the values
		sort.Slice(item.Values, func(i, j int) bool {
			return item.Values[i] < item.Values[j]
		})

		for _, v := range item.Values {
			// deduping the values
			candidate := iface.ExternalAPIGetValue{
				Value: v,
				Cloud: item.Cloud,
			}
			if s.isGetItemValueUnique(res.Values, candidate) {
				res.Values = append(res.Values, candidate)
			}
		}
	}

	return &res, nil
}

func (s *MetadataService) isGetItemValueUnique(values []iface.ExternalAPIGetValue, candidate iface.ExternalAPIGetValue) bool {
	for _, v := range values {
		if v.Cloud == candidate.Cloud && v.Value == candidate.Value {
			return false
		}
	}

	return true
}

func (s *MetadataService) isListItemUnique(values []iface.ExternalAPIListItem, candidate iface.ExternalAPIListItem) bool {
	for _, v := range values {
		if v.ID == candidate.ID && v.Type == candidate.Type && v.Label == candidate.Label {
			return false
		}
	}

	return true
}

// since the optional items stored differently, we need to map them to the correct type.
func (s *MetadataService) mapListItemOptionals(listsByType map[metadata.MetadataFieldType][]metadataDalIface.ListItem) {
	// adding the optionals to the lists
	for _, optionalItem := range listsByType[metadata.MetadataFieldTypeOptional] {
		for _, value := range optionalItem.Values {
			mappped := metadataDalIface.ListItem{
				Key:       value,
				Type:      optionalItem.SubType,
				Label:     gcpMetadataDomain.FormatLabel(value, optionalItem.SubType),
				Timestamp: optionalItem.Timestamp,
			}

			listsByType[optionalItem.SubType] = append(listsByType[optionalItem.SubType], mappped)
		}
	}

	listsByType[metadata.MetadataFieldTypeOptional] = nil
}

func (s *MetadataService) filterAttrGroups(attrGroups []*domain.OrgMetadataModel, keyFilter string, typeFilter string) []metadataDalIface.GetItem {
	filteredList := make([]metadataDalIface.GetItem, 0)

	for _, item := range attrGroups {
		if item.Key == keyFilter && string(item.Type) == typeFilter {
			newItem := metadataDalIface.GetItem{
				Key:    item.Key,
				Type:   item.Type,
				Label:  item.Label,
				Values: item.Values,
				Cloud:  item.Cloud,
			}
			if newItem.Values == nil {
				newItem.Values = []string{}
			}

			filteredList = append(filteredList, newItem)
		}
	}

	return filteredList
}

// UpdateAllCustomersMetadata updates all customers' Azure metadata.
func (s *MetadataService) UpdateAzureAllCustomersMetadata(ctx context.Context) ([]error, error) {
	return s.azureMetadata.UpdateAllCustomersMetadata(ctx)
}

// UpdateAzureCustomerMetadata updates a customer's Azure metadata for all organizations.
func (s *MetadataService) UpdateAzureCustomerMetadata(ctx context.Context, customerID string) error {
	return s.azureMetadata.UpdateCustomerMetadata(ctx, customerID, "")
}

// UpdateAzureCustomerOrganizationMetadata updates a customer's Azure metadata for a specific organization.
func (s *MetadataService) UpdateAzureCustomerOrganizationMetadata(ctx context.Context, customerID, organizationID string) error {
	return s.azureMetadata.UpdateCustomerMetadata(ctx, customerID, organizationID)
}

// UpdateBQLensAllCustomersMetadata updates all customers' BQLens metadata.
func (s *MetadataService) UpdateBQLensAllCustomersMetadata(ctx context.Context) ([]error, error) {
	return s.bqlensMetadata.UpdateAllCustomersMetadata(ctx)
}

// UpdateBQLensCustomerMetadata updates a customer's BQLens metadata for all organizations.
func (s *MetadataService) UpdateBQLensCustomerMetadata(ctx context.Context, customerID string) error {
	return s.bqlensMetadata.UpdateCustomerMetadata(ctx, customerID, "")
}

// UpdateBQLensCustomerOrganizationMetadata updates a customer's BQLens metadata for a specific organization.
func (s *MetadataService) UpdateBQLensCustomerOrganizationMetadata(ctx context.Context, customerID, organizationID string) error {
	return s.bqlensMetadata.UpdateCustomerMetadata(ctx, customerID, organizationID)
}

// UpdateBillingAccountMetadata updates a GCP billing account's metadata for one or more organizations.
func (s *MetadataService) UpdateGCPBillingAccountMetadata(ctx context.Context, assetID, billingAccountID string, orgs []*common.Organization) error {
	return s.gcpMetadata.UpdateBillingAccountMetadata(ctx, assetID, billingAccountID, orgs)
}

// UpdateAWSCustomersMetadata updates all customers' AWS metadata.
func (s *MetadataService) UpdateAWSAllCustomersMetadata(ctx context.Context) ([]error, error) {
	return s.awsMetadata.UpdateAllCustomersMetadata(ctx)
}

// UpdateAWSCustomerMetadata updates a customer's AWS metadata for all organizations.
func (s *MetadataService) UpdateAWSCustomerMetadata(ctx context.Context, customerID string, orgs []*common.Organization) error {
	return s.awsMetadata.UpdateCustomerMetadata(ctx, customerID, orgs)
}

// UpdateDataHubMetadata updates the metadata with the events stored in the GCS bucket.
func (s *MetadataService) UpdateDataHubMetadata(ctx context.Context) error {
	return s.datahubMetadata.UpdateDataHubMetadata(ctx)
}
