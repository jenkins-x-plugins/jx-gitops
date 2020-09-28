package variablefinders

import (
	"os"
)

//  FindPipelineKind finds the pipeline kind
func FindPipelineKind() (string, error) {
	jobType := os.Getenv("JOB_TYPE")
	prNumber := os.Getenv("PULL_NUMBER")
	if jobType == "presubmit" || prNumber != "" {
		return "pullrequest", nil
	}
	return "release", nil
}
