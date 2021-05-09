package helmfiles_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles"
	"github.com/stretchr/testify/assert"
)

func TestSplitChartName(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input           string
		expectPrefix    string
		expectLocalName string
	}{
		{
			input:           "localOnly",
			expectPrefix:    "",
			expectLocalName: "localOnly",
		},
		{
			input:           "jx3/lighthouse",
			expectPrefix:    "jx3",
			expectLocalName: "lighthouse",
		},
	}

	for _, tc := range testCases {
		prefix, localName := helmfiles.SpitChartName(tc.input)
		assert.Equal(t, tc.expectPrefix, prefix, "prefix for input %s", tc.input)
		assert.Equal(t, tc.expectLocalName, localName, "localName for input %s", tc.input)
	}

}
