package variables_test

import (
	"io/ioutil"
	"strings"
	"testing"

	"os"
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/pr/variables"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/fakerunners"
	"github.com/jenkins-x/go-scm/scm"
	scmfake "github.com/jenkins-x/go-scm/scm/driver/fake"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = false
)

func TestPullRequestVariables(t *testing.T) {
	// lets skip this test if inside a goreleaser when we've got the env vars defined
	chartEnv := os.Getenv("JX_CHART_REPOSITORY")
	if chartEnv != "" {
		t.Skipf("skipping test as $JX_CHART_REPOSITORY = %s\n", chartEnv)
		return
	}

	prNumber := 123
	repo := "myorg/myrepo"
	prBranch := "my-pr-branch-name"
	expectedHeadClone := "https://github.com/jenkins-x-labs-bot/myrepo.git"

	fakePR := &scm.PullRequest{
		Number: prNumber,
		Title:  "my awesome pull request",
		Body:   "some text",
		Source: prBranch,
		Base: scm.PullRequestBranch{
			Ref: "main",
			Sha: "fakesha1234",
		},
		Head: scm.PullRequestBranch{
			Ref: "my-branch",
			Sha: "fakesha5678",
			Repo: scm.Repository{
				Clone: expectedHeadClone,
			},
		},
		Labels: []*scm.Label{
			{
				Name: "updatebot",
			},
			{
				Name: "env/staging",
			},
			{
				Name: "some:label",
			},
		},
	}

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create temp dir")

	testDir := filepath.Join("test_data")
	fs, err := ioutil.ReadDir(testDir)
	require.NoError(t, err, "failed to read test dir %s", testDir)
	for _, f := range fs {
		if f == nil || !f.IsDir() {
			continue
		}
		name := f.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		srcDir := filepath.Join(testDir, name)
		runDir := filepath.Join(tmpDir, name)

		err := files.CopyDirOverwrite(srcDir, runDir)
		require.NoError(t, err, "failed to copy from %s to %s", srcDir, runDir)

		t.Logf("running test %s in dir %s\n", name, runDir)

		version := "1.2.3"
		versionFile := filepath.Join(runDir, "VERSION")
		err = ioutil.WriteFile(versionFile, []byte(version), files.DefaultFileWritePermissions)
		require.NoError(t, err, "failed to write file %s", versionFile)

		ns := "jx"
		devEnv := jxenv.CreateDefaultDevEnvironment(ns)
		devEnv.Namespace = ns
		devEnv.Spec.Source.URL = "https://github.com/jx3-gitops-repositories/jx3-kubernetes.git"
		if name == "nokube" {
			devEnv.Spec.Source.URL = "https://github.com/jx3-gitops-repositories/jx3-github.git"
		} else {
			requirements := jxcore.NewRequirementsConfig()
			requirements.Spec.Cluster.ChartRepository = "http://bucketrepo/bucketrepo/charts/"
			data, err := yaml.Marshal(requirements)
			require.NoError(t, err, "failed to marshal requirements")
			devEnv.Spec.TeamSettings.BootRequirements = string(data)
		}

		runner := fakerunners.NewFakeRunnerWithGitClone()

		jxClient := jxfake.NewSimpleClientset(devEnv)
		scmClient, fakeData := scmfake.NewDefault()
		fakeData.PullRequests[prNumber] = fakePR

		_, o := variables.NewCmdPullRequestVariables()

		o.Dir = runDir
		o.CommandRunner = runner.Run
		o.JXClient = jxClient
		o.Namespace = ns

		o.Options.Owner = "MyOwner"
		o.Options.Repository = "myrepo"
		o.Options.Branch = "PR-23"
		o.Options.SourceURL = "https://github.com/" + repo

		o.CommandRunner = runner.Run
		o.SourceURL = "https://github.com/" + repo
		o.Number = prNumber
		o.Branch = prBranch
		o.ScmClient = scmClient

		if name == "comments" {
			o.UseComments = true

			fakeData.PullRequestComments = map[int][]*scm.Comment{
				prNumber: {
					{
						ID:   1,
						Body: "some text\n/jx-var FOO=bar\n/jx-var CHEESE = edam\n\nsomething",
					},
					{
						ID:   2,
						Body: "/jx-var FOO=newValue",
					},
					{
						ID:   3,
						Body: ` /jx-var WITH_QUOTES = " some value " `,
					},
				},
			}
		}

		err = o.Run()

		require.NoError(t, err, "failed to run the command")

		f := filepath.Join(runDir, o.File)
		require.FileExists(t, f, "should have generated file")
		t.Logf("generated file %s\n", f)

		expectedPath := filepath.Join(srcDir, "expected.sh")
		if generateTestOutput {
			generatedFile := f
			data, err := ioutil.ReadFile(generatedFile)
			require.NoError(t, err, "failed to load %s", generatedFile)

			err = ioutil.WriteFile(expectedPath, data, 0600)
			require.NoError(t, err, "failed to save file %s", expectedPath)

			t.Logf("saved file %s\n", expectedPath)
			continue
		}

		testhelpers.AssertTextFilesEqual(t, expectedPath, f, "generated file")
	}

}
