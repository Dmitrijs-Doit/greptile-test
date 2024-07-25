package googleclouddirect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchBillingAccount(t *testing.T) {
	v, _ := MatchBillingAccount("ABCABC-ABC123-ABC583")
	assert.True(t, v)

	v, _ = MatchBillingAccount("ABCABC-ABC123-ABC58345")
	assert.False(t, v)

	v, _ = MatchBillingAccount("ABCABC-ABC123-ABC58Z")
	assert.False(t, v)
}

func TestMatchProject(t *testing.T) {
	v, _ := MatchProject("how-are-you-22")
	assert.True(t, v)

	v, _ = MatchProject("2how-are-you")
	assert.False(t, v)

	v, _ = MatchProject("How-are-you")
	assert.False(t, v)
}

func TestMatchDataset(t *testing.T) {
	v, _ := MatchDataset("XxXx_65")
	assert.True(t, v)

	v, _ = MatchDataset("hello-world")
	assert.False(t, v)
}
