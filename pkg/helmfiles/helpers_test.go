package helmfiles_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles"
	"github.com/stretchr/testify/assert"
)

func TestGatherHelmfiles(t *testing.T) {
	expected := []helmfiles.Helmfile{
		{
			Filepath:           "helpers_test_data/helmfile.yaml",
			RelativePathToRoot: "",
		},
		{
			Filepath:           "helpers_test_data/helmfiles/jx/helmfile.yaml",
			RelativePathToRoot: "../../",
		},
		{
			Filepath:           "helpers_test_data/helmfiles/jx/helmfiles/nested/helmfile.yaml",
			RelativePathToRoot: "../../../../",
		},
		{
			Filepath:           "helpers_test_data/helmfiles/nginx/helmfile.yaml",
			RelativePathToRoot: "../../",
		},
		{
			Filepath:           "helpers_test_data/helmfiles/cert-manager/helmfile.yaml",
			RelativePathToRoot: "../../",
		},
	}

	actual, err := helmfiles.GatherHelmfiles("helmfile.yaml", "helpers_test_data")
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
