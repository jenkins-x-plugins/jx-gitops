package filters_test

import (
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/filters"
	"github.com/stretchr/testify/assert"
)

func TestStringFilters(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		filter   filters.StringFilter
		value    string
		expected bool
	}{
		{
			filter: filters.StringFilter{
				Prefix: "Merge pull request",
			},
			value:    "Merge pull request #1234",
			expected: true,
		},
		{
			filter: filters.StringFilter{
				Prefix: "Merge pull request",
			},
			value:    "something else",
			expected: false,
		},
		{
			filter: filters.StringFilter{
				Prefix: "!Merge pull request",
			},
			value:    "Merge pull request #1234",
			expected: false,
		},
		{
			filter: filters.StringFilter{
				Prefix: "!Merge pull request",
			},
			value:    "something else",
			expected: true,
		},
		{
			filter: filters.StringFilter{
				Suffix: "awesome",
			},
			value:    "something else awesome",
			expected: true,
		},
		{
			filter: filters.StringFilter{
				Suffix: "awesome",
			},
			value:    "something else",
			expected: false,
		},
		{
			filter: filters.StringFilter{
				Contains: "awesome",
			},
			value:    "something awesome thingy",
			expected: true,
		},
		{
			filter: filters.StringFilter{
				Contains: "awesome",
			},
			value:    "something thingy",
			expected: false,
		},
		{
			filter: filters.StringFilter{
				Contains: "!awesome",
			},
			value:    "something awesome thingy",
			expected: false,
		},
		{
			filter: filters.StringFilter{
				Contains: "!awesome",
			},
			value:    "something thingy",
			expected: true,
		},
		{
			filter: filters.StringFilter{
				Prefix:   "Merge pull request",
				Contains: "cool",
				Suffix:   "awesome",
			},
			value:    "Merge pull request very cool awesome",
			expected: true,
		},
		{
			filter: filters.StringFilter{
				Prefix:   "Merge pull request",
				Contains: "cool",
				Suffix:   "awesome",
			},
			value:    "Merge pull request awesome",
			expected: false,
		},
	}

	for _, tc := range testCases {
		actual := tc.filter.Matches(tc.value)
		message := tc.filter.String()
		assert.Equal(t, tc.expected, actual, "for value %s with filter %s", tc.value, message)
		t.Logf("filter %s matched: %v for input: %s", message, actual, tc.value)
	}
}
