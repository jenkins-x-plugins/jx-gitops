package export_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/export"
	jenkinsio "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io"
	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestExportRepositorySourceDir(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	_, o := export.NewCmdExportConfig()
	ns := "jx"
	o.Namespace = ns
	o.JXClient = fake.NewSimpleClientset(
		createGitHubSourceRepository(ns, "jenkins-x", "jx-cli"),
		createGitHubSourceRepository(ns, "jenkins-x", "jx-gitops"),
	)
	generatedFile := filepath.Join(tmpDir, v1alpha1.SourceConfigFileName)
	o.ConfigFile = generatedFile

	err = o.Run()
	require.NoError(t, err, "failed to run the export")

	t.Logf("generated export file %s", o.ConfigFile)

	testhelpers.AssertTextFilesEqual(t, filepath.Join("test_data", "expected.yaml"), generatedFile, "generated source config file")
}

func createGitHubSourceRepository(ns, org, repo string) *jenkinsv1.SourceRepository {
	return &jenkinsv1.SourceRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SourceRepository",
			APIVersion: jenkinsio.GroupName + "/" + jenkinsio.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      org + "-" + repo,
			Namespace: ns,
		},
		Spec: jenkinsv1.SourceRepositorySpec{
			Provider:     "https://github.com",
			Org:          org,
			Repo:         repo,
			ProviderName: "github",
			Scheduler: jenkinsv1.ResourceReference{
				Name: "cheese",
			},
		},
	}
}
