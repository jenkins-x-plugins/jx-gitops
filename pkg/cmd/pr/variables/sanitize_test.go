package variables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSanitizeLabelName(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple label",
			input:    "bug",
			expected: "BUG",
		},
		{
			name:     "label with dash",
			input:    "feature-request",
			expected: "FEATURE_REQUEST",
		},
		{
			name:     "label with slash",
			input:    "env/staging",
			expected: "ENV_STAGING",
		},
		{
			name:     "label with colon",
			input:    "some:label",
			expected: "SOME_LABEL",
		},
		{
			name:     "label with multiple special chars",
			input:    "priority: high",
			expected: "PRIORITY_HIGH",
		},
		{
			name:     "label with multiple consecutive special chars",
			input:    "feature---request",
			expected: "FEATURE_REQUEST",
		},
		{
			name:     "label with mixed special chars",
			input:    "my_-_label",
			expected: "MY_LABEL",
		},
		{
			name:     "label with emoji",
			input:    "🐛 bug",
			expected: "BUG",
		},
		{
			name:     "label with emoji only",
			input:    "🎉",
			expected: "",
		},
		{
			name:     "label with leading/trailing special chars",
			input:    "---test---",
			expected: "TEST",
		},
		{
			name:     "label with leading/trailing underscores",
			input:    "___test___",
			expected: "TEST",
		},
		{
			name:     "label with spaces",
			input:    "needs review",
			expected: "NEEDS_REVIEW",
		},
		{
			name:     "label with multiple spaces",
			input:    "needs    review",
			expected: "NEEDS_REVIEW",
		},
		{
			name:     "label with numbers",
			input:    "v2.0",
			expected: "V2_0",
		},
		{
			name:     "label starting with number",
			input:    "2fa-enabled",
			expected: "2FA_ENABLED",
		},
		{
			name:     "label with parentheses",
			input:    "wip(feature)",
			expected: "WIP_FEATURE",
		},
		{
			name:     "empty label",
			input:    "",
			expected: "",
		},
		{
			name:     "label with only special chars",
			input:    "---",
			expected: "",
		},
		{
			name:     "existing underscore preserved",
			input:    "my_label",
			expected: "MY_LABEL",
		},
		{
			name:     "mixed underscores and dashes",
			input:    "my_-label",
			expected: "MY_LABEL",
		},
		{
			name:     "updatebot label",
			input:    "updatebot",
			expected: "UPDATEBOT",
		},
		{
			name:     "env/staging label",
			input:    "env/staging",
			expected: "ENV_STAGING",
		},
		{
			name:     "some:label",
			input:    "some:label",
			expected: "SOME_LABEL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeLabelName(tc.input)
			require.Equal(t, tc.expected, result)
		})
	}
}
