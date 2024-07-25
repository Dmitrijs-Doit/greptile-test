package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegionsProcessedCorretlyForMasterPayerAccountsValueAll(t *testing.T) {
	payer := &MasterPayerAccount{
		AccountNumber: "111",
		Regions:       []string{"all"},
		TenancyType:   "dedicated",
	}

	regions := payer.GetMasterPayerAccountRegions()

	assert.True(t, RegionsArrayContainsAllValue(regions))
	assert.True(t, payer.IsValidRegion("af-south-1"))
	assert.True(t, payer.IsValidRegion("us-east-1"))
	assert.True(t, payer.IsValidRegion("ap-southeast-2"))
	assert.False(t, payer.IsValidRegion("some-region"))
}

func TestEmptyRegionsProcessedCorrectlyForSharedPayerAccounts(t *testing.T) {
	payer := &MasterPayerAccount{
		AccountNumber: "222",
		Regions:       []string{},
		TenancyType:   "shared",
	}

	regions := payer.GetMasterPayerAccountRegions()

	assert.True(t, len(regions) == 15)
	assert.True(t, payer.IsValidRegion("ap-southeast-2"))
	assert.False(t, payer.IsValidRegion("af-south-1"))
	assert.False(t, payer.IsValidRegion("some-region"))
}

func TestExtraRegionsFetchedCorretlyForDedicatedMasterPayerAccounts(t *testing.T) {
	payer := &MasterPayerAccount{
		AccountNumber: "333",
		Regions:       []string{"some-region"},
		TenancyType:   "dedicated",
	}

	regions := payer.GetMasterPayerAccountRegions()

	assert.True(t, len(regions) == 8)
	assert.True(t, payer.IsValidRegion("ap-northeast-1"))
	assert.True(t, payer.IsValidRegion("us-east-1"))
	assert.True(t, payer.IsValidRegion("some-region"))
	assert.False(t, payer.IsValidRegion("af-south-1"))
	assert.False(t, payer.IsValidRegion("eu-west-2"))
	assert.False(t, payer.IsValidRegion("some-other-region"))
}
