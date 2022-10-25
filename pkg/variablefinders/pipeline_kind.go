package variablefinders

import (
	"os"
	"strings"
)

// FindPipelineKind finds the pipeline kind
func FindPipelineKind(branch string) (string, error) {
	jobType := os.Getenv("JOB_TYPE")
	prNumber := os.Getenv("PULL_NUMBER")
	if strings.HasPrefix(branch, "PR-") || jobType == "presubmit" || prNumber != "" {
		return "pullrequest", nil
	}
	return "release", nil
}
