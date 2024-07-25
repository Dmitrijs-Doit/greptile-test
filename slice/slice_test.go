package slice

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestFindIndex tests the FindIndex function
func TestFindIndex(t *testing.T) {
	var findIndexPayloads = []struct {
		slice    []string
		item     string
		expected int
	}{
		{[]string{"a", "b", "c"}, "a", 0},
		{[]string{"a", "b", "c"}, "b", 1},
		{[]string{"a", "b", "c"}, "d", -1},
	}

	for _, p := range findIndexPayloads {
		assert.Equal(t, p.expected, FindIndex(p.slice, p.item))
	}
}

// TestContains tests the Contains function
func TestContains(t *testing.T) {
	var containsPayloads = []struct {
		slice    []string
		item     string
		expected bool
	}{
		{[]string{"a", "b", "c"}, "a", true},
		{[]string{"a", "b", "c"}, "b", true},
		{[]string{"a", "b", "c"}, "d", false},
	}

	for _, p := range containsPayloads {
		assert.Equal(t, p.expected, Contains(p.slice, p.item))
	}
}

// TestContainsAny tests the ContainsAny function
func TestContainsAny(t *testing.T) {
	var containsAnyPayloads = []struct {
		slice    []string
		items    []string
		expected bool
	}{
		{[]string{"a", "b", "c"}, []string{"a"}, true},
		{[]string{"a", "b", "c"}, []string{"d", "b", "a"}, true},
		{[]string{"a", "b", "c"}, []string{"d", "e", "f"}, false},
	}

	for _, p := range containsAnyPayloads {
		assert.Equal(t, p.expected, ContainsAny(p.slice, p.items))
	}
}

func TestUnorderedEquality(t *testing.T) {
	var equalPayloads = []struct {
		first, second, separator string
	}{
		{"now,lets,see,if,this,works", "lets,now,if,this,works,see", ","},
		{"Cloud Natural Language;Geocoding API;Maps API;Translate;Places API;Cloud Storage;Directions API;Maps Static API;Cloud AutoML;Cloud DNS;Cloud Dialogflow API",
			"Directions API;Geocoding API;Maps API;Translate;Cloud Natural Language;Places API;Cloud Storage;Maps Static API;Cloud Dialogflow API;Cloud AutoML;Cloud DNS", ";"},
		{"", "", ","},
		{"Well, Just a single item", "Well, Just a single item", ";"},
		{strings.Join([]string{"What if, separator is part of the item", "it should still work", "even if,it is not a proper separator", "well, lets hopt it works"}, ", "),
			strings.Join([]string{"it should still work", "What if, separator is part of the item", "well, lets hopt it works", "even if,it is not a proper separator"}, ", "), ", "},
		{"test,test,string", "test,string,test", ","},
	}

	for _, p := range equalPayloads {
		assert.True(t, UnorderedSeparatedStringsComp(p.first, p.second, p.separator))
	}
}

func TestUnorderedInequality(t *testing.T) {
	var unequalPayloads = []struct {
		first, second, separator string
	}{
		{"this,should,not,work", "this,shouldn't,not,work", ","},
		{"Cloud Natural Language;Geocoding API;Maps API;Translate;Places API;Cloud Storage;Directions API;Maps Static API;Cloud AutoML;Cloud DNS;Cloud Dialogflow API", "", ";"},
		{"", "Cloud Natural Language;Geocoding API;Maps API;Translate;Places API;Cloud Storage;Directions API;Maps Static API;Cloud AutoML;Cloud DNS;Cloud Dialogflow API", ";"},
		{"Not, equal, number, of, elements", "elements, equal, number, of", ", "},
		{"This,is,not,invariant,comparison", "this,is,not,invariant,comparison", ","},
		{"test,test,string", "test,nice,string", ","},
		{"test,nice,string", "test,test,string", ","},
	}

	for _, p := range unequalPayloads {
		assert.False(t, UnorderedSeparatedStringsComp(p.first, p.second, p.separator))
	}
}

func TestContainsSubAt(t *testing.T) {
	var containsSubAtPayloads = []struct {
		slice    []string
		sub      string
		expected int
	}{
		{[]string{"abc", "def", "ghi"}, "de", 1},
		{[]string{"abc", "def", "ghi"}, "hi", 2},
		{[]string{"abc", "def", "ghi"}, "xy", -1},
		{[]string{"abc", "def", "ghi"}, "abc", 0},
	}

	for _, p := range containsSubAtPayloads {
		assert.Equal(t, p.expected, ContainsSubAt(p.slice, p.sub))
	}
}
