package build_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helm/build"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/require"
)

func TestStepHelmBuildWithCharts(t *testing.T) {
	sourceData := "testdata"

	path := filepath.Join(sourceData, "has_charts")

	runner := &fakerunner.FakeRunner{}
	helmBin := "helm"

	_, o := build.NewCmdHelmBuild()
	o.HelmBinary = helmBin
	o.CommandRunner = runner.Run
	o.ChartsDir = filepath.Join(path, "charts")

	err := o.Run()
	require.NoError(t, err, "failed to run the command")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "helm repo add 0 file://myapp-common",
		},

		fakerunner.FakeResult{
			CLI: "helm lint",
		},
		fakerunner.FakeResult{
			CLI: "helm dependency build .",
		},
		fakerunner.FakeResult{
			CLI: "helm package .",
		},
	)

}

func TestStepHelmBuildWithChartsOCI(t *testing.T) {
	sourceData := "testdata"

	path := filepath.Join(sourceData, "has_charts")

	runner := &fakerunner.FakeRunner{}
	helmBin := "helm"

	_, o := build.NewCmdHelmBuild()
	o.HelmBinary = helmBin
	o.CommandRunner = runner.Run
	o.ChartsDir = filepath.Join(path, "charts")
	o.OCI = true

	err := o.Run()
	require.NoError(t, err, "failed to run the command")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "helm repo add 0 file://myapp-common",
		},
		fakerunner.FakeResult{
			CLI: "helm lint",
		},
		fakerunner.FakeResult{
			CLI: "helm dependency build . --registry-config " + o.RegistryConfigFile,
		},
		fakerunner.FakeResult{
			CLI: "helm package . --registry-config " + o.RegistryConfigFile,
		},
	)

}

func TestStepHelmBuildWithChartsOCIPassword(t *testing.T) {
	sourceData := "testdata"

	path := filepath.Join(sourceData, "has_charts")

	runner := &fakerunner.FakeRunner{}
	helmBin := "helm"

	_, o := build.NewCmdHelmBuild()
	o.HelmBinary = helmBin
	o.CommandRunner = runner.Run
	o.ChartsDir = filepath.Join(path, "charts")
	o.OCI = true
	o.RepositoryPassword = "xxx"
	o.RepositoryUsername = "fish"

	err := o.Run()
	require.NoError(t, err, "failed to run the command")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "helm repo add 0 file://myapp-common",
		},
		fakerunner.FakeResult{
			CLI: "helm lint",
		},
		fakerunner.FakeResult{
			CLI: "helm registry login  --username fish --password xxx",
		},

		fakerunner.FakeResult{
			CLI: "helm dependency build .",
		},
		fakerunner.FakeResult{
			CLI: "helm package .",
		},
	)

}

func TestStepHelmBuildWithNoCharts(t *testing.T) {
	sourceData := "testdata"

	path := filepath.Join(sourceData, "no_charts")

	runner := &fakerunner.FakeRunner{}
	helmBin := "helm"

	_, o := build.NewCmdHelmBuild()
	o.HelmBinary = helmBin
	o.CommandRunner = runner.Run
	o.ChartsDir = filepath.Join(path, "charts")

	err := o.Run()
	require.NoError(t, err, "failed to run the command")
	fmt.Printf("runner.ResultOutput: %v\n", runner.ResultOutput)

}
