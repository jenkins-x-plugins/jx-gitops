package variables

import (
	"strings"
)

func sanitizeLabelName(label string) string {
	sanitized := strings.ToUpper(label)

	var result strings.Builder
	result.Grow(len(sanitized))

	var lastCharWasUnderscore bool
	for _, char := range sanitized {
		if (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			result.WriteRune(char)
			lastCharWasUnderscore = false
		} else if !lastCharWasUnderscore {
			result.WriteRune('_')
			lastCharWasUnderscore = true
		}
	}

	return strings.Trim(result.String(), "_")
}
