package flexsaveresold

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudhealth"
	"github.com/stretchr/testify/assert"
)

const delta = 1e-10

func TestDistributeAutopilotUtilizationPerHourAllocatesUtilization(t *testing.T) {
	testService, _, _, testLog1, err := createTestService("dummy_project", nil)
	if err != nil {
		assert.Fail(t, "test failed, could not create testContext")
	}

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 7, 23, 59, 999, 999999, time.UTC)

	order := newFlexRIOrder(5001, "us-east-2", "t3.large", "Linux/Unix", 2,
		startDate, endDate, 0.0200, 0.0150, 0.0050)

	var report cloudhealth.SingleDimensionReport

	data, _ := os.ReadFile("./testjsons/autopilotTestSimpleDistributionAndCosts.json_doNotFormat")

	json.Unmarshal(data, &report)

	mtdTimeInstance := time.Date(2025, 5, 7, 8, 0, 0, 0, time.UTC)
	apOrders := make([]*FlexRIOrder, 0)
	apOrders = append(apOrders, order)

	emptyUsage := false
	if len(report.Data) == 0 {
		emptyUsage = true
	}

	distributionAttributes := UtilizationDistributionAttributes{
		mtdTimeInstance: mtdTimeInstance,
		autopilotOrders: apOrders,
		log:             testLog1,
		groupKey:        "xx",
	}
	distributeAutopilotUtilizationPerHour(&distributionAttributes, report, emptyUsage)

	utilization := order.Autopilot

	assert.NotNil(t, testService)
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-01"]["00"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-01"]["01"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-01"]["02"])
	assert.Equal(t, 6.0, utilization.Utilization["2025-01-01"]["11"])
	assert.Equal(t, 6.0, utilization.Utilization["2025-01-01"]["12"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-02"]["00"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-02"]["01"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-02"]["02"])
	assert.Equal(t, 6.0, utilization.Utilization["2025-01-02"]["11"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-03"]["00"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-03"]["01"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-03"]["02"])
	assert.Equal(t, 6.0, utilization.Utilization["2025-01-03"]["11"])
	assert.Equal(t, 4.0, utilization.Utilization["2025-01-04"]["15"])
}

func TestDistributionOverlaysOnPreviousUtilization(t *testing.T) {
	testService, _, _, testLog1, err := createTestService("dummy_project", nil)
	if err != nil {
		assert.Fail(t, "test failed, could not create testContext")
	}

	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 7, 23, 59, 999, 999999, time.UTC)

	order := newFlexRIOrder(5001, "us-east-2", "t3.large", "Linux/Unix", 2,
		startDate, endDate, 0.0200, 0.0150, 0.0050)
	order.Autopilot = &FlexRIAutopilot{}
	order.Autopilot.Utilization = mallocMapOfMapOfFloats()
	utilization := order.Autopilot

	utilization.Utilization["2025-01-01"] = mallocMapOfFloatsIfAbsent("2025-01-01", order.Autopilot.Utilization)
	utilization.Utilization["2025-01-02"] = mallocMapOfFloatsIfAbsent("2025-01-02", order.Autopilot.Utilization)

	rand.Seed(time.Now().UnixNano())
	hr0, hr1, hr2, hr11, hr12 := rand.Float64()*8, rand.Float64()*8, rand.Float64()*8, rand.Float64()*8, rand.Float64()*8
	utilization.Utilization["2025-01-01"]["00"] = hr0
	utilization.Utilization["2025-01-01"]["01"] = hr1
	utilization.Utilization["2025-01-01"]["02"] = hr2
	utilization.Utilization["2025-01-01"]["11"] = hr11
	utilization.Utilization["2025-01-01"]["12"] = hr12
	utilization.Utilization["2025-01-02"]["00"] = rand.Float64() * 100 // doesn't matter, will be overlaid
	utilization.Utilization["2025-01-02"]["01"] = rand.Float64() * 100 // doesn't matter, will be overlaid

	var report cloudhealth.SingleDimensionReport

	data, _ := os.ReadFile("./testjsons/autopilotTestPreviousSnapshotIsOverlaid.json_doNotFormat")
	json.Unmarshal(data, &report)

	mtdTimeInstance := time.Date(2025, 5, 7, 8, 0, 0, 0, time.UTC)
	apOrders := make([]*FlexRIOrder, 0)
	apOrders = append(apOrders, order)
	reportHrs := report.Dimensions[0]["time"]
	reportData := report.Data

	emptyUsage := false
	if len(report.Data) == 0 {
		emptyUsage = true
	}

	distributionAttributes := UtilizationDistributionAttributes{
		mtdTimeInstance: mtdTimeInstance,
		autopilotOrders: apOrders,
		log:             testLog1,
		groupKey:        "xx",
	}

	distributeAutopilotUtilizationPerHour(&distributionAttributes, report, emptyUsage)

	assert.NotNil(t, testService)
	assert.Equal(t, hr0, utilization.Utilization["2025-01-01"]["00"])
	assert.Equal(t, hr1, utilization.Utilization["2025-01-01"]["01"])
	assert.Equal(t, hr2, utilization.Utilization["2025-01-01"]["02"])
	assert.Equal(t, hr11, utilization.Utilization["2025-01-01"]["11"])
	assert.Equal(t, hr12, utilization.Utilization["2025-01-01"]["12"])
	assert.Equal(t, "2025-01-02 00:00", reportHrs[1].Name) // 0th reportHrs is total
	assert.Equal(t, "2025-01-02 01:00", reportHrs[2].Name)
	assert.Equal(t, "2025-01-02 02:00", reportHrs[3].Name)
	assert.Equal(t, "2025-01-02 11:00", reportHrs[4].Name)
	assert.Equal(t, "2025-01-03 00:00", reportHrs[5].Name)
	assert.Equal(t, "2025-01-03 01:00", reportHrs[6].Name)
	assert.Equal(t, "2025-01-03 02:00", reportHrs[7].Name)
	assert.Equal(t, "2025-01-03 11:00", reportHrs[8].Name)
	assert.Equal(t, "2025-01-04 15:00", reportHrs[9].Name)
	assert.Equal(t, 10, len(reportHrs))
	assert.Equal(t, 10, len(reportData))

	assert.Equal(t, 8.0, utilization.Utilization["2025-01-02"]["00"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-02"]["01"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-02"]["02"])
	assert.Equal(t, 6.0, utilization.Utilization["2025-01-02"]["11"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-03"]["00"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-03"]["01"])
	assert.Equal(t, 8.0, utilization.Utilization["2025-01-03"]["02"])
	assert.Equal(t, 6.0, utilization.Utilization["2025-01-03"]["11"])
	assert.Equal(t, 4.0, utilization.Utilization["2025-01-04"]["15"])
}

func TestSpecialCaseNanoAndSmallOrderWithoutMacro(t *testing.T) {
	testService, _, _, testLog1, err := createTestService("dummy_project", nil)
	if err != nil {
		assert.Fail(t, "test failed, could not create testContext")
	}

	someMonth := time.January
	startDay := 1
	lengthOfMonth, lengthOfDay := 7, 5 // 7 day month, 5 hour day
	startDate := time.Date(2025, someMonth, startDay, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, someMonth, lengthOfMonth+1, 0, 0, 0, 0, time.UTC).Add(-1)
	mtdTimeInstance := monthToDateTimeStamp(time.Date(2025, 1, 7, 8, 0, 0, 0, time.UTC))
	onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized := 500000.0200, 0.0150, 0.0050 // onDemandValue is irrelevant, only for reference

	sfOrder1 := newFlexRIOrder(5001, "us-east-2", "t3.nano", "Linux/Unix", 1,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized)
	sfOrder1.NormalizedUnits.UnitsPerDay = sfOrder1.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in
	sfOrder2 := newFlexRIOrder(5002, "us-east-2", "t3.small", "Linux/Unix", 1,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized) // already normalized!
	sfOrder2.NormalizedUnits.UnitsPerDay = sfOrder2.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in

	apOrders := []*FlexRIOrder{sfOrder1, sfOrder2}

	var report cloudhealth.SingleDimensionReport

	data, _ := os.ReadFile("./testjsons/autopilotTestSimpleDistributionAndCosts.json_doNotFormat")
	json.Unmarshal(data, &report)

	emptyUsage := false
	if len(report.Data) == 0 {
		emptyUsage = true
	}

	distributionAttributes := UtilizationDistributionAttributes{
		mtdTimeInstance: mtdTimeInstance,
		autopilotOrders: apOrders,
		log:             testLog1,
		groupKey:        "xx",
	}

	distributeAutopilotUtilizationPerHour(&distributionAttributes, report, emptyUsage)
	calculateAutopilotQualifiedCosts(apOrders, mtdTimeInstance, float64(lengthOfDay))

	util1 := sfOrder1.Autopilot
	util2 := sfOrder2.Autopilot

	assert.NotNil(t, testService)
	assert.Equal(t, 0.25, util1.Utilization["2025-01-01"]["00"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-01"]["01"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-01"]["02"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-01"]["11"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-01"]["12"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-02"]["00"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-02"]["01"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-02"]["02"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-02"]["11"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-03"]["00"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-03"]["01"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-03"]["02"])
	assert.Equal(t, 0.25, util1.Utilization["2025-01-03"]["11"])
	assert.Equal(t, 0.00, util1.Utilization["2025-01-03"]["15"]) // outside mtdTimeStamp-2days-extrahrs

	assert.Equal(t, 1.00, util2.Utilization["2025-01-01"]["00"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-01"]["01"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-01"]["02"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-01"]["11"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-01"]["12"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-02"]["00"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-02"]["01"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-02"]["02"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-02"]["11"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-03"]["00"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-03"]["01"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-03"]["02"])
	assert.Equal(t, 1.00, util2.Utilization["2025-01-03"]["11"])
	assert.Equal(t, 0.00, util2.Utilization["2025-01-03"]["15"]) // outside mtdTimeStamp-2days-extrahrs

	assert.Equal(t, util1.MTDQualifiedLineUnits, 1.0) // smallest order size allocated first, all(1) recommended lineUnits generated savings
	assert.Equal(t, util1.MTDQualifiedUtilization, 3.25)
	assert.Equal(t, util1.MTDApSavingsAtFlexRIRate, 3.25*savingsPerHourNormalized) // (5, 4, 4)*0.25 on 1st,2nd & 3rd day used
	assert.Equal(t, util1.MTDApPenaltyAtFlexRIRate, 0.5*flexRIPerHourNormalized)   // 1 unit * 0.25 on 2nd & 3rd day wasted
	assert.Equal(t, util1.MTDUnqualifiedUtilization, float64(0))                   // for sf1 no units were discarded

	assert.Equal(t, util1.MTDQualifiedLineUnits, 1.0) // of remaining 1 out of 1 possible lineUnits,
	assert.Equal(t, util2.MTDQualifiedUtilization, 13.00)
	assert.Equal(t, util2.MTDApSavingsAtFlexRIRate, 13*savingsPerHourNormalized) // 5, 4, 4 on 1st,2nd & 3rd day used
	assert.Equal(t, util2.MTDApPenaltyAtFlexRIRate, 2*flexRIPerHourNormalized)   // 1 unit each on 2nd & 3rd day wasted
	assert.Equal(t, util2.MTDUnqualifiedUtilization, float64(0))                 // for sf1 no units were discarded
}

func TestCalculateAutopilotQualifiedCosts(t *testing.T) {
	testService, _, _, testLog1, err := createTestService("dummy_project", nil)
	if err != nil {
		assert.Fail(t, "test failed, could not create testContext")
	}

	someMonth := time.January
	startDay := 1
	lengthOfMonth, lengthOfDay := 7, 5 // 7 day month
	startDate := time.Date(2025, someMonth, startDay, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, someMonth, lengthOfMonth+1, 0, 0, 0, 0, time.UTC).Add(-1)
	mtdTimeInstance := monthToDateTimeStamp(time.Date(2025, 1, 7, 8, 0, 0, 0, time.UTC))
	onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized := 500000.0200, 0.0150, 0.0050 // onDemandValue is irrelevant, only for reference

	sfOrder1 := newFlexRIOrder(5001, "us-east-2", "t3.small", "Linux/Unix", 2,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized)
	sfOrder1.NormalizedUnits.UnitsPerDay = sfOrder1.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in
	sfOrder2 := newFlexRIOrder(5002, "us-east-2", "t3.large", "Linux/Unix", 2,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized) // already normalized!
	sfOrder2.NormalizedUnits.UnitsPerDay = sfOrder2.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in

	apOrders := make([]*FlexRIOrder, 0)
	apOrders = append(apOrders, sfOrder1)
	apOrders = append(apOrders, sfOrder2)

	var report cloudhealth.SingleDimensionReport

	data, _ := os.ReadFile("./testjsons/autopilotTestSimpleDistributionAndCosts.json_doNotFormat")
	json.Unmarshal(data, &report)

	emptyUsage := false
	if len(report.Data) == 0 {
		emptyUsage = true
	}

	distributionAttributes := UtilizationDistributionAttributes{
		mtdTimeInstance: mtdTimeInstance,
		autopilotOrders: apOrders,
		log:             testLog1,
		groupKey:        "xx",
	}

	distributeAutopilotUtilizationPerHour(&distributionAttributes, report, emptyUsage)
	calculateAutopilotQualifiedCosts(apOrders, mtdTimeInstance, float64(lengthOfDay))

	util1 := sfOrder1.Autopilot
	util2 := sfOrder2.Autopilot

	assert.NotNil(t, testService)
	assert.Equal(t, 2.0, util1.MTDQualifiedLineUnits) // smallest order size allocated first, all(2) recommended lineUnits generated savings
	assert.Equal(t, util1.MTDQualifiedUtilization, float64(26))
	assert.Equal(t, util1.MTDApSavingsAtFlexRIRate, 26*savingsPerHourNormalized) // (5+5, 4+4, 4+4) on 1st,2nd & 3rd day
	assert.Equal(t, util1.MTDApPenaltyAtFlexRIRate, 4*flexRIPerHourNormalized)   // (1+1, 1+1) wasted on 2nd & 3rd day
	assert.Equal(t, util1.MTDUnqualifiedUtilization, float64(0))                 // for sf1 no units were discarded

	assert.Equal(t, 4.0, util2.MTDQualifiedLineUnits)           // of remaining, 4 lineUnits generated savings
	assert.Equal(t, util2.MTDQualifiedUtilization, float64(52)) // only 4 rows qualified (13*4) across 1st,2nd & 3rd day
	assert.Equal(t, util2.MTDApSavingsAtFlexRIRate, 52*savingsPerHourNormalized)
	assert.Equal(t, util2.MTDApPenaltyAtFlexRIRate, 8*flexRIPerHourNormalized)
	assert.Equal(t, util2.MTDUnqualifiedUtilization, float64(18)) //within eligible, 8 nfu discarded
}

func TestMetalTypesAreHandledCorrectlyForAutopilotQualifiedCosts(t *testing.T) {
	testService, _, _, testLog1, err := createTestService("dummy_project", nil)
	if err != nil {
		assert.Fail(t, "test failed, could not create testContext")
	}

	someMonth := time.January
	startDay := 1
	lengthOfMonth, lengthOfDay := 7, 5 // 7 day month
	startDate := time.Date(2025, someMonth, startDay, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, someMonth, lengthOfMonth+1, 0, 0, 0, 0, time.UTC).Add(-1)
	mtdTimeInstance := monthToDateTimeStamp(time.Date(2025, 1, 7, 8, 0, 0, 0, time.UTC))
	onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized := 500000.0200, 0.0150, 0.0050 // onDemandValue is irrelevant, only for reference

	sfOrder1 := newFlexRIOrder(5001, "us-east-2", "r6g.metal", "Linux/Unix", 1,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized)
	sfOrder1.NormalizedUnits.UnitsPerDay = sfOrder1.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in
	sfOrder2 := newFlexRIOrder(5002, "us-east-2", "r6g.large", "Linux/Unix", 40,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized) // already normalized!
	sfOrder2.NormalizedUnits.UnitsPerDay = sfOrder2.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in

	apOrders := make([]*FlexRIOrder, 0)
	apOrders = append(apOrders, sfOrder1)
	apOrders = append(apOrders, sfOrder2)

	var report cloudhealth.SingleDimensionReport

	data, _ := os.ReadFile("./testjsons/autopilotTestMetalsDistributionAndCosts.json_doNotFormat")
	json.Unmarshal(data, &report)

	emptyUsage := false
	if len(report.Data) == 0 {
		emptyUsage = true
	}

	distributionAttributes := UtilizationDistributionAttributes{
		mtdTimeInstance: mtdTimeInstance,
		autopilotOrders: apOrders,
		log:             testLog1,
		groupKey:        "xx",
	}

	distributeAutopilotUtilizationPerHour(&distributionAttributes, report, emptyUsage)
	calculateAutopilotQualifiedCosts(apOrders, mtdTimeInstance, float64(lengthOfDay))

	util1 := sfOrder1.Autopilot // metal (bubbled up as smaller per hour units)
	util2 := sfOrder2.Autopilot // large

	assert.NotNil(t, testService)

	assert.Equal(t, util1.MTDQualifiedUtilization, float64(1664))
	assert.InDelta(t, util1.MTDApSavingsAtFlexRIRate, 1664*savingsPerHourNormalized, delta) // (128*5, 128*4, 128*4) on 1st,2nd & 3rd day
	assert.InDelta(t, util1.MTDApPenaltyAtFlexRIRate, 256*flexRIPerHourNormalized, delta)   // (128, 128) wasted on 2nd & 3rd day
	assert.Equal(t, util1.MTDUnqualifiedUtilization, float64(0))                            // for sf1 no units were discarded

	assert.Equal(t, util2.MTDQualifiedUtilization, float64(806))                             // (62*5. 62*4, 62*4)
	assert.InDelta(t, util2.MTDApSavingsAtFlexRIRate, 62*13*savingsPerHourNormalized, delta) // 62 lines qualified, each having used 13 units
	assert.InDelta(t, util2.MTDApPenaltyAtFlexRIRate, 62*2*flexRIPerHourNormalized, delta)   // (1 unit each(total 2) on day 2,3 wasted for 62 lines)
	assert.Equal(t, util2.MTDUnqualifiedUtilization, float64(98*9))                          //9 units in each of 98 lines discarded (9 units of 384 where usage was left)
}

func TestCalculateAutopilotQualifiedCostsSampleCase(t *testing.T) {
	testService, _, _, testLog1, err := createTestService("dummy_project", nil)
	if err != nil {
		assert.Fail(t, "test failed, could not create testContext")
	}

	someMonth := time.April
	startDay := 1
	lengthOfMonth, lengthOfDay := 30, 24 // standard day month
	startDate := time.Date(2025, someMonth, startDay, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, someMonth, lengthOfMonth+1, 0, 0, 0, 0, time.UTC).Add(-1)
	mtdTimeInstance := monthToDateTimeStamp(time.Date(2025, 5, 4, 8, 0, 0, 0, time.UTC))
	onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized := 0.0208, 0.01536, 0.0054 // onDemandValue is irrelevant, only for reference

	sfOrder1 := newFlexRIOrder(5001, "us-east-2", "t3.small", "Linux/Unix", 1,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized)
	sfOrder1.NormalizedUnits.UnitsPerDay = sfOrder1.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in
	sfOrder2 := newFlexRIOrder(5002, "us-east-2", "t3.medium", "Linux/Unix", 1,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized) // already normalized!
	sfOrder2.NormalizedUnits.UnitsPerDay = sfOrder2.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in
	sfOrder3 := newFlexRIOrder(5003, "us-east-2", "t3.large", "Linux/Unix", 1,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized) // already normalized!
	sfOrder3.NormalizedUnits.UnitsPerDay = sfOrder3.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in

	apOrders := make([]*FlexRIOrder, 0)
	apOrders = append(apOrders, sfOrder1)
	apOrders = append(apOrders, sfOrder2)
	apOrders = append(apOrders, sfOrder3)

	var report cloudhealth.SingleDimensionReport

	data, _ := os.ReadFile("./testjsons/autopilotSampleRequestTest.json_doNotFormat")
	json.Unmarshal(data, &report)

	emptyUsage := false
	if len(report.Data) == 0 {
		emptyUsage = true
	}

	distributionAttributes := UtilizationDistributionAttributes{
		mtdTimeInstance: mtdTimeInstance,
		autopilotOrders: apOrders,
		log:             testLog1,
		groupKey:        "xx",
	}

	distributeAutopilotUtilizationPerHour(&distributionAttributes, report, emptyUsage)
	calculateAutopilotQualifiedCosts(apOrders, mtdTimeInstance, float64(lengthOfDay))

	assert.NotNil(t, testService)

	util1, util2, util3 := sfOrder1.Autopilot, sfOrder2.Autopilot, sfOrder3.Autopilot

	assert.Equal(t, util1.MTDQualifiedLineUnits, 1.0) // smallest order size allocated first, all(1) recommended lineUnits generated savings
	assert.InDelta(t, 696.0, util1.MTDQualifiedUtilization, delta)
	assert.InDelta(t, 3.75840, util1.MTDApSavingsAtFlexRIRate, delta)
	assert.InDelta(t, 0.368640, util1.MTDApPenaltyAtFlexRIRate, delta)
	util1TotalAutopilotSavings := util1.MTDApSavingsAtFlexRIRate - util1.MTDApPenaltyAtFlexRIRate
	assert.InDelta(t, 3.389760, util1TotalAutopilotSavings, delta)

	assert.Equal(t, 0.0, util1.MTDUnqualifiedUtilization)
	assert.Equal(t, 0.0, util1.MTDApSavingsForDiscardedUsageAtFlexRIRate)
	assert.Equal(t, 0.0, util1.MTDApPenaltyForDiscardedUsageAtFlexRIRate)

	assert.Equal(t, util2.MTDQualifiedLineUnits, 2.0) // of remaining, 2 lineUnits generated savings
	assert.InDelta(t, 1392.00, util2.MTDQualifiedUtilization, delta)
	assert.InDelta(t, 7.51680, util2.MTDApSavingsAtFlexRIRate, delta)
	assert.InDelta(t, 0.737280, util2.MTDApPenaltyAtFlexRIRate, delta)
	util2TotalAutopilotSavings := util2.MTDApSavingsAtFlexRIRate - util2.MTDApPenaltyAtFlexRIRate
	assert.InDelta(t, 6.779520, util2TotalAutopilotSavings, delta)

	assert.Equal(t, 0.0, util2.MTDUnqualifiedUtilization)
	assert.Equal(t, 0.0, util2.MTDApSavingsForDiscardedUsageAtFlexRIRate)
	assert.Equal(t, 0.0, util2.MTDApPenaltyForDiscardedUsageAtFlexRIRate)

	assert.Equal(t, util3.MTDQualifiedLineUnits, 3.0) // of remaining, 3 lineUnits generated savings
	assert.InDelta(t, 1992.00, util3.MTDQualifiedUtilization, delta)
	assert.InDelta(t, 10.75680, util3.MTDApSavingsAtFlexRIRate, delta)
	assert.InDelta(t, 2.580480, util3.MTDApPenaltyAtFlexRIRate, delta)
	util3TotalAutopilotSavings := util3.MTDApSavingsAtFlexRIRate - util3.MTDApPenaltyAtFlexRIRate
	assert.InDelta(t, 8.176320, util3TotalAutopilotSavings, delta)

	assert.InDelta(t, 360.0, util3.MTDUnqualifiedUtilization, delta)
	assert.InDelta(t, 1.9440, util3.MTDApSavingsForDiscardedUsageAtFlexRIRate, delta)
	assert.InDelta(t, 5.52960, util3.MTDApPenaltyForDiscardedUsageAtFlexRIRate, delta)

	assert.InDelta(t, 18.34560, util1TotalAutopilotSavings+util2TotalAutopilotSavings+util3TotalAutopilotSavings, delta)
}

func TestTimeCalculations(t *testing.T) {
	testTime1Feb25 := time.Date(2025, 02, 2, 12, 30, 0, 0, time.UTC)
	_2DaysBeforetestTime2Feb25 := testTime1Feb25.UTC().Truncate(time.Hour * 24)
	_2DaysBeforetestTime2Feb25 = _2DaysBeforetestTime2Feb25.Add(time.Hour * -48)
	assert.Equal(t, _2DaysBeforetestTime2Feb25, time.Date(2025, 01, 31, 00, 00, 0, 0, time.UTC))
	fmt.Println(_2DaysBeforetestTime2Feb25)
}

func TestFindIncrementAndIterations(t *testing.T) {
	large_2 := 8.0
	largeFactor := 4.0
	iterations, increment := findIncrementAndIterations(large_2, largeFactor)
	assert.Equal(t, increment, float64(1))
	assert.Equal(t, iterations, int64(8))

	xlarge_2 := 16.0
	xlargeFactor := 8.0
	iterations, increment = findIncrementAndIterations(xlarge_2, xlargeFactor)
	assert.Equal(t, increment, float64(1))
	assert.Equal(t, iterations, int64(16))

	micro_3 := 0.75
	microFactor := 0.25
	iterations, increment = findIncrementAndIterations(micro_3, microFactor)
	assert.Equal(t, increment, float64(0.25))
	assert.Equal(t, iterations, int64(3))

	micro_7 := 1.75
	iterations, increment = findIncrementAndIterations(micro_7, microFactor)
	assert.Equal(t, increment, float64(0.25))
	assert.Equal(t, iterations, int64(7))

	small_1 := 0.50
	smallFactor := 0.50
	iterations, increment = findIncrementAndIterations(small_1, smallFactor)
	assert.Equal(t, increment, float64(0.50))
	assert.Equal(t, iterations, int64(1))
}

func TestFlexRIOrderReducer(t *testing.T) {
	someMonth := time.April
	startDay := 1
	lengthOfMonth, lengthOfDay := 30, 24 // standard day month
	startDate := time.Date(2025, someMonth, startDay, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, someMonth, lengthOfMonth+1, 0, 0, 0, 0, time.UTC).Add(-1)
	onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized := 0.0208, 0.01536, 0.0054 // onDemandValue is irrelevant, only for reference

	sfOrder1 := newFlexRIOrder(5001, "us-east-2", "t3.small", "Linux/Unix", 1,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized)
	sfOrder1.NormalizedUnits.UnitsPerDay = sfOrder1.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in
	sfOrder2 := newFlexRIOrder(5002, "us-east-2", "t3.medium", "Linux/Unix", 1,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized) // already normalized!
	sfOrder2.NormalizedUnits.UnitsPerDay = sfOrder2.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in
	sfOrder3 := newFlexRIOrder(5003, "us-east-2", "t3.large", "Linux/Unix", 1,
		startDate, endDate, onDemandPerHourNormalized, flexRIPerHourNormalized, savingsPerHourNormalized) // already normalized!
	sfOrder3.NormalizedUnits.UnitsPerDay = sfOrder3.NormalizedUnits.UnitsPerHour * float64(lengthOfDay) // wire this one in

	apOrders := make([]*FlexRIOrder, 0)
	apOrders = append(apOrders, sfOrder1)
	apOrders = append(apOrders, sfOrder2)
	apOrders = append(apOrders, sfOrder3)

	orderDetails := flexRIOrderReducer(apOrders, flexRIOrderToIdsMapper)
	assert.Equal(t, "5001, 5002, 5003", orderDetails)
}

func newFlexRIOrder(orderId int64, region, instanceType, operatingSystem string,
	numberOfInstances int64, startdate, enddate time.Time, ondemandNormRate, flexRINormRate, SavingPerHourNorm float64) *FlexRIOrder {
	addrOfString := func(s string) *string { return &s }
	//addrOfInt64 := func(i int64) *int64 { return &i }
	//addrOfFloat64 := func(i float64) *float64 { return &i }
	//addrOfTime := func(i time.Time) *time.Time { return &i }
	addrOfBool := func(i bool) *bool { return &i }

	instance := strings.SplitN(instanceType, ".", 2)

	_, normalizedSize, _ := InstanceFamilyNormalizationFactor(instanceType)

	order := FlexRIOrder{
		ID:       orderId,
		Customer: nil,
		Entity:   nil,
		Status:   OrderStatusNew,
		Email:    "testing@test.test",
		UID:      "foofle:123",
		Config: FlexRIOrderConfig{
			Region:          &region,
			InstanceType:    &instanceType,
			InstanceFamily:  &instance[0],
			OperatingSystem: &operatingSystem,
			Tenancy:         addrOfString("default"),
			NumInstances:    &numberOfInstances,
			StartDate:       &startdate,
			EndDate:         &enddate,
			SizeFlexible:    addrOfBool(true),
		},
		ClientID:           -999,
		InvoiceAdjustments: FlexRIOrderInvoiceAdjustments{},
		NormalizedUnits: &FlexRIOrderNormalizedUnits{
			Factor:       normalizedSize,
			UnitsPerHour: normalizedSize * float64(numberOfInstances),
		},
		Pricing: &FlexRIOrderPricing{
			OnDemandNormalized:       &ondemandNormRate,
			FlexibleNormalized:       &flexRINormRate,
			SavingsPerHourNormalized: &SavingPerHourNorm,
		},
		Utilization: map[string]float64{},
		Metadata: map[string]interface{}{
			"customer": map[string]interface{}{
				"primaryDomain": "customerPrimaryDomain",
				"name":          "customerName",
			},
		},
	}

	return &order
}

func TestDistributeEmptyUsageToAutopilotUtilization(t *testing.T) {
	testService, _, _, testLog1, err := createTestService("dummy_project", nil)
	if err != nil {
		assert.Fail(t, "test failed, could not create testContext")
	}

	startDate := time.Date(2025, 02, 01, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 03, 01, 0, 0, 0, 0, time.UTC).Add(-1) // endDate = 2025-02-28 23:59:999
	nowTime := time.Date(2025, 02, 25, 18, 0, 0, 0, time.UTC)
	order := newFlexRIOrder(5001, "us-east-2", "t3.large", "Linux/Unix", 2,
		startDate, endDate, 0.0200, 0.0150, 0.0050)

	var report cloudhealth.SingleDimensionReport

	data, _ := os.ReadFile("./testjsons/autopilotTestEmptyUsage.json_doNotFormat")
	json.Unmarshal(data, &report)

	apOrders := make([]*FlexRIOrder, 0)
	apOrders = append(apOrders, order)

	mtdTimeInstance := monthToDateTimeStamp(nowTime)
	distributionAttributes := UtilizationDistributionAttributes{
		mtdTimeInstance: mtdTimeInstance,
		nowTime:         nowTime,
		autopilotOrders: apOrders,
		log:             testLog1,
		groupKey:        "xx",
	}

	emptyUsage := false
	if len(report.Data) == 0 {
		emptyUsage = true
	}

	distributeAutopilotUtilizationPerHour(&distributionAttributes, report, emptyUsage)

	utilization := order.Autopilot

	assert.NotNil(t, testService)
	assert.Equal(t, 21, len(utilization.Utilization))
	assert.Equal(t, 24, len(utilization.Utilization["2025-02-15"]))
	assert.Equal(t, 0.0, utilization.Utilization["2025-02-02"]["02"])
	assert.Equal(t, 0.0, utilization.Utilization["2025-02-10"]["11"])
	assert.Equal(t, 0.0, utilization.Utilization["2025-02-17"]["15"])
}

// For days 1-3 after of the month after startDate
func TestDistributeEmptyUsageForDaysAfterEndDate(t *testing.T) {
	testService, _, _, testLog1, err := createTestService("dummy_project", nil)
	if err != nil {
		assert.Fail(t, "test failed, could not create testContext")
	}

	startDate := time.Date(2025, 02, 01, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 03, 01, 0, 0, 0, 0, time.UTC).Add(-1) // endDate = 2025-02-28 23:59:999
	nowTime := time.Date(2025, 03, 04, 18, 0, 0, 0, time.UTC)
	order := newFlexRIOrder(5001, "us-east-2", "t3.large", "Linux/Unix", 2,
		startDate, endDate, 0.0200, 0.0150, 0.0050)

	var report cloudhealth.SingleDimensionReport

	data, _ := os.ReadFile("./testjsons/autopilotTestEmptyUsage.json_doNotFormat")
	json.Unmarshal(data, &report)

	apOrders := make([]*FlexRIOrder, 0)
	apOrders = append(apOrders, order)

	mtdTimeInstance := monthToDateTimeStamp(nowTime)
	distributionAttributes := UtilizationDistributionAttributes{
		mtdTimeInstance: mtdTimeInstance,
		nowTime:         nowTime,
		autopilotOrders: apOrders,
		log:             testLog1,
		groupKey:        "xx",
	}

	emptyUsage := false
	if len(report.Data) == 0 {
		emptyUsage = true
	}

	distributeAutopilotUtilizationPerHour(&distributionAttributes, report, emptyUsage)

	utilization := order.Autopilot

	assert.NotNil(t, testService)
	assert.Equal(t, 28, len(utilization.Utilization))
	assert.Equal(t, 24, len(utilization.Utilization["2025-02-15"]))
	assert.Equal(t, 0.0, utilization.Utilization["2025-02-02"]["04"])
	assert.Equal(t, 0.0, utilization.Utilization["2025-02-10"]["16"])
	assert.Equal(t, 0.0, utilization.Utilization["2025-02-28"]["21"])
}
