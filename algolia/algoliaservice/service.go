package algoliaservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/opt"
	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/doitintl/hello/scheduled-tasks/algolia"
	"github.com/doitintl/hello/scheduled-tasks/algolia/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Service struct {
	loggerProvider logger.ILogger

	userDAL     dal.UserDAL
	customerDAL customerDal.Customers
	algoliaDAL  dal.AlgoliaDAL

	Config *algolia.Config
}

func NewAlgoliaService(log logger.Provider, conn *connection.Connection) (*Service, error) {
	ctx := context.Background()

	algoliaDAL := dal.NewAlgoliaFirestoreWithClient(conn.Firestore)
	config, err := algoliaDAL.GetConfigFromFirestore(ctx)

	if err != nil {
		return nil, err
	}

	userDAL := dal.NewUserFirestore(conn)
	customersDAL := customerDal.NewCustomersFirestoreWithClient(conn.Firestore)

	return &Service{
		loggerProvider: log(ctx),
		userDAL:        userDAL,
		customerDAL:    customersDAL,
		algoliaDAL:     algoliaDAL,
		Config:         config,
	}, nil
}

func (s *Service) GetAPIKey(ctx context.Context, customerID string, userID string) (*algolia.Config, error) {
	user, err := s.userDAL.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	allowed := []string{"alerts", "routes"}
	restricted := []string{"customers"}

	var noAccessControlIndices = []string{"alerts", "routes"}

	permissionChecks := []struct {
		hasAccessControl bool
		Index            []string
		Check            func(context.Context, *common.User) bool
	}{
		{false, []string{"users"}, s.userDAL.HasUsersPermission},
		{false, []string{"invoices"}, s.userDAL.HasInvoicesPermission},
		{false, []string{"assets"}, s.userDAL.HasLicenseManagePermission},
		{false, []string{"entities"}, s.userDAL.HasEntitiesPermission},
		{true, []string{"reports"}, s.userDAL.HasCloudAnalyticsPermission},
		{true, []string{"attributions", "attributionGroups"}, s.userDAL.HasAttributionsPermission},
		{true, []string{"budgets"}, s.userDAL.HasBudgetsPermission},
		{false, []string{"metrics"}, s.userDAL.HasMetricsPermission},
	}

	for _, p := range permissionChecks {
		if p.Check(ctx, user) {
			allowed = append(allowed, p.Index...)

			if !p.hasAccessControl {
				noAccessControlIndices = append(noAccessControlIndices, p.Index...)
			}

			continue
		}

		restricted = append(restricted, p.Index...)
	}

	//Filter restrictions: https://www.algolia.com/doc/api-reference/api-parameters/filters/
	/**
	We start from the premise that only cloud analytics indices have public and collaborators fields.
	"noAccessControlIndices" is needed in order to have a valid right hand condition for the filter (collaborators and public are not present in indices which don't have access control)
	*/

	filter := fmt.Sprintf("(customerId:%s OR type:preset) AND (collaborators.email:%s OR public:viewer OR public:editor OR _indexName:%s)", customerID, user.Email, strings.Join(noAccessControlIndices, " OR _indexName:"))

	validUntil := time.Now().Add(time.Hour * 24)
	key, err := search.GenerateSecuredAPIKey(s.Config.SearchKey, opt.Filters(filter), opt.RestrictIndices(allowed...), opt.ValidUntil(validUntil), opt.UserToken(userID))

	if err != nil {
		return nil, err
	}

	return &algolia.Config{
		AppID:             s.Config.AppID,
		SearchKey:         key,
		RestrictedIndices: restricted,
	}, nil
}
