package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoveTrailingSlashes(t *testing.T) {
	type inputOutput struct {
		input  string
		output string
	}

	tests := []inputOutput{
		{
			input:  "/mr_slashy_at_the_beginning_sentence",
			output: "mr_slashy_at_the_beginning_sentence",
		},
		{
			input:  "mr_slashy_at_the_end/",
			output: "mr_slashy_at_the_end",
		},
		{
			input:  "/mr_slashy_two_ways/",
			output: "mr_slashy_two_ways",
		},
		{
			input:  "/m/r/_/s/l/a///shy_/two_ways/",
			output: "m/r/_/s/l/a///shy_/two_ways",
		},
		{
			input:  "mr_no_slashes",
			output: "mr_no_slashes",
		},
		{
			input:  "",
			output: "",
		},
	}

	for _, test := range tests {
		res := RemoveLeadingAndTrailingSlashes(test.input)
		assert.Equal(t, test.output, res)
	}
}
