package scripts

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/spot0/api/model"
)

const csvFilePref = "/tmp/fbod_report"

type ResponseItem struct {
	PrimaryDomain         string
	CustomerID            string
	AsgName               string
	IsLaunchTemplate      bool
	IsLaunchConfiguration bool
	IsMip                 bool
	IsAdvancedSpotOption  bool
}

type ResponseItems []*ResponseItem

func (r ResponseItems) Len() int           { return len(r) }
func (r ResponseItems) Less(i, j int) bool { return r[i].PrimaryDomain < r[j].PrimaryDomain }
func (r ResponseItems) Swap(i, j int)      { r[i], r[j] = r[j], r[i] }

type FbodStatisticsParams struct {
	CustomerID string `json:"customer_id,omitempty"`
}

func FbodStatistics(ctx *gin.Context) []error {
	var params FbodStatisticsParams

	if err := ctx.ShouldBindJSON(&params); err != nil {
		// if nothing is passed as param it returns error and that's OK
		fmt.Printf("error parsing params: %s", err)
	}

	fs, err := firestore.NewClient(ctx, "me-doit-intl-com")
	if err != nil {
		return []error{err}
	}

	query := fs.Collection("spot0").Doc("spotApp").Collection("asgs").
		Where("config.fallbackOnDemand", "==", true)

	if params.CustomerID != "" {
		customerRef := fs.Collection("customers").Doc(params.CustomerID)
		query = query.Where("customer", "==", customerRef)
	}

	asgs, err := query.Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	var errors []error

	var (
		responseItems     []*ResponseItem
		noLTItems         []*ResponseItem
		lcItems           []*ResponseItem
		advancedSpotItems []*ResponseItem
		noMipItems        []*ResponseItem
		mipItems          []*ResponseItem
	)

	common.RunConcurrentJobsOnCollection(ctx, asgs, 5, func(ctx context.Context, asgSnap *firestore.DocumentSnapshot) {
		parseAsgConf(ctx, asgSnap, &responseItems, &noLTItems, &lcItems, &noMipItems, &advancedSpotItems, &mipItems, &errors)
	})

	errors = append(errors, writeCSV(responseItems, "all_asgs"))
	errors = append(errors, writeCSV(noLTItems, "no_lt_items"))
	errors = append(errors, writeCSV(lcItems, "lc_items"))
	errors = append(errors, writeCSV(advancedSpotItems, "advancedSpotItems"))
	errors = append(errors, writeCSV(noMipItems, "no_mip_items"))
	errors = append(errors, writeCSV(mipItems, "mip_items"))

	return errors
}

func parseAsgConf(ctx context.Context, asgSnap *firestore.DocumentSnapshot,
	responseItems, noLTItems, lcItems, noMipItems, advancedSpotItems, mipItems *[]*ResponseItem,
	errors *[]error) {
	var asg model.AsgConfiguration
	if err := asgSnap.DataTo(&asg); err != nil {
		fmt.Printf("error parsing asg: %s", err)
		*errors = append(*errors, err)
	}

	customerRef := asg.Customer.(*firestore.DocumentRef)
	customer, err := common.GetCustomer(ctx, customerRef)

	if err != nil {
		*errors = append(*errors, err)
	}

	isLaunchTemplate := asg.Spotisize.CurAsg.LaunchTemplate != nil || asg.Spotisize.CurAsg.MixedInstancesPolicy != nil
	isLaunchConfiguration := asg.Spotisize.CurAsg.LaunchConfigurationName != nil
	isMip := asg.Spotisize.CurAsg.MixedInstancesPolicy != nil
	isAdvancedSpotOption := strings.Contains(asg.SpotisizeErrorDesc, "already requesting spot instances")

	responseItem := ResponseItem{
		customer.PrimaryDomain,
		customerRef.ID,
		asg.AsgName,
		isLaunchTemplate,
		isLaunchConfiguration,
		isMip,
		isAdvancedSpotOption,
	}

	*responseItems = append(*responseItems, &responseItem)

	if !isMip && !isLaunchConfiguration {
		*noMipItems = append(*noMipItems, &responseItem)
	}

	if isLaunchConfiguration {
		*lcItems = append(*lcItems, &responseItem)
	}

	if !isLaunchTemplate {
		*noLTItems = append(*noLTItems, &responseItem)
	}

	if isAdvancedSpotOption {
		*advancedSpotItems = append(*advancedSpotItems, &responseItem)
	}

	if isMip {
		*mipItems = append(*mipItems, &responseItem)
	}
}

func writeCSV(responseItems ResponseItems, name string) error {
	outFile, err := os.Create(csvFilePref + "_" + name + ".csv")
	if err != nil {
		return err
	}

	sort.Sort(responseItems)

	records := [][]string{
		{
			"PrimaryDomain",
			"CustomerID",
			"AsgName",
			"IsLaunchTemplate",
			"IsLaunchConfiguration",
			"IsMip",
			"IsAdvancedSpotOption",
		},
	}

	for _, item := range responseItems {
		var record []string
		record = append(record, item.PrimaryDomain)
		record = append(record, item.CustomerID)
		record = append(record, item.AsgName)
		record = append(record, strconv.FormatBool(item.IsLaunchTemplate))
		record = append(record, strconv.FormatBool(item.IsLaunchConfiguration))
		record = append(record, strconv.FormatBool(item.IsMip))
		record = append(record, strconv.FormatBool(item.IsAdvancedSpotOption))
		records = append(records, record)
	}

	w := csv.NewWriter(outFile)

	return w.WriteAll(records)
}
