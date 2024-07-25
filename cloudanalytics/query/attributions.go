package query

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"cloud.google.com/go/bigquery"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain"
	attrDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/google/uuid"
)

type Query struct {
	bq               *bigquery.Client
	attributionQuery iface.IAttributionQuery
}

func NewQuery(bq *bigquery.Client) *Query {
	return &Query{
		bq:               bq,
		attributionQuery: NewAttributionQuery(),
	}
}

func (q *Query) HandleAttributions(ctx context.Context, fp *domain.AttrFiltersParams, attributions []*queryDomain.QueryRequestX) error {
	for outerI, attr := range attributions {
		predicates := make([]string, 0)

		for i, attrRow := range attr.Composite {
			id := fmt.Sprintf("%s:%s:%d", attr.ID, attrRow.Key, i)

			switch attrRow.Type {
			case metadata.MetadataFieldTypeFixed:
				filter, queryParam, err := GetFixedFilter(attrRow, id, fmt.Sprint(outerI))
				if err != nil {
					return err
				}

				if filter != "" {
					predicates = append(predicates, filter)
				}

				if queryParam != nil {
					fp.QueryParams = append(fp.QueryParams, *queryParam)
				}

			case metadata.MetadataFieldTypeLabel,
				metadata.MetadataFieldTypeTag,
				metadata.MetadataFieldTypeProjectLabel,
				metadata.MetadataFieldTypeSystemLabel,
				metadata.MetadataFieldTypeGKELabel:
				filter, queryParam, err := GetTagLabelFilter(attrRow, id)
				if err != nil {
					return err
				}

				if filter != "" {
					predicates = append(predicates, filter)
				}

				if queryParam != nil {
					fp.QueryParams = append(fp.QueryParams, *queryParam)
				}
			case metadata.MetadataFieldTypeGKE:
				filter, queryParam, err := GetFixedFilter(attrRow, id, fmt.Sprint(outerI))
				if err != nil {
					return err
				}

				if filter != "" {
					predicates = append(predicates, filter)
				}

				if queryParam != nil {
					fp.QueryParams = append(fp.QueryParams, *queryParam)
				}
			default:
			}
		}

		if len(predicates) > 0 {
			compositeFilter, err := q.buildCompositeFilter(ctx, attr, predicates)
			if err != nil {
				return err
			}

			attributionID := attr.ID
			if attributionID == "" {
				attributionID = uuid.New().String()
			}

			attributionNameParam := base64.RawStdEncoding.EncodeToString([]byte(attributionID))
			queryParam := bigquery.QueryParameter{
				Name:  attributionNameParam,
				Value: attr.Key,
			}
			fp.QueryParams = append(fp.QueryParams, queryParam)

			fp.AttrConditions = append(fp.AttrConditions, fmt.Sprintf(`(IF(%s, @%s, NULL))`, compositeFilter, attributionNameParam))
			fp.MetricFilters = append(fp.MetricFilters, fmt.Sprintf(`%s AS %c`, compositeFilter, consts.ASCIIAInt+outerI))

			if attr.IncludeInFilter {
				fp.CompositeFilters = append(fp.CompositeFilters, fmt.Sprintf("(%s)", compositeFilter))
			}
		}
	}

	return nil
}

func (q *Query) buildCompositeFilter(ctx context.Context, attr *queryDomain.QueryRequestX, predicates []string) (string, error) {
	// Formula must bet set.
	if attr.Formula == "" {
		return "", ErrEmptyFormula
	}

	if err := q.attributionQuery.ValidateFormula(ctx, q.bq, len(predicates), attr.Formula); err != nil {
		return "", err
	}

	compositeFilter := attr.Formula
	// clean operators out of formula string to allow for proper replacement
	compositeFilter = q.attributionQuery.LogicalOperatorsAlphaToSymbol(compositeFilter)

	var replacements []string
	for i := 0; i < len(predicates); i++ {
		replacements = append(replacements, getVariableStringFromIndex(i), predicates[i])
	}

	compositeFilter = strings.NewReplacer(replacements...).Replace(compositeFilter)
	// restore operators to formula string
	return q.attributionQuery.LogicalOperatorsSymbolToAlpha(compositeFilter), nil
}

func (q *Query) GetOrgsAttributionsQuery(ctx context.Context, fp *domain.AttrFiltersParams, org *common.Organization) error {
	var attributions []*queryDomain.QueryRequestX

	for _, attrRef := range org.Scope {
		attrSnap, err := attrRef.Get(ctx)
		if err != nil {
			switch status.Code(err) {
			case codes.NotFound:
				return web.ErrNotFound
			case codes.PermissionDenied:
				return web.ErrForbidden
			default:
				return web.ErrInternalServerError
			}
		}

		var attr attrDomain.Attribution
		if err := attrSnap.DataTo(&attr); err != nil {
			return err
		}

		attr.ID = attrRef.ID
		attributions = append(attributions, attr.ToQueryRequestX(true))
	}

	if err := q.HandleAttributions(ctx, fp, attributions); err != nil {
		return err
	}

	return nil
}
