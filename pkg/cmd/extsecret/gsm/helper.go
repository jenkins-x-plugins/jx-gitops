package gsm

import (
	"strings"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
)

// GetCurrentGCPProject returns the current GCP project ID
func GetCurrentGCPProject() (string, error) {
	cmd := cmdrunner.Command{
		Name: "gcloud",
		Args: []string{"config", "get-value", "project"},
	}
	out, err := cmd.RunWithoutRetry()
	if err != nil {
		return "", err
	}

	index := strings.LastIndex(out, "\n")
	if index >= 0 {
		return out[index+1:], nil
	}

	return out, nil
}
