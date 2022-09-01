//go:build unit

package pipelinescheduler_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/pipelinescheduler"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/pipelinescheduler/testhelpers"
	"github.com/stretchr/testify/assert"

	"github.com/pborman/uuid"
)

func TestBuild(t *testing.T) {
	org := uuid.New()
	leaf1 := &pipelinescheduler.SchedulerLeaf{
		Org:           org,
		Repo:          uuid.New(),
		SchedulerSpec: testhelpers.CompleteScheduler(),
	}
	leaf2 := &pipelinescheduler.SchedulerLeaf{
		Org:           org,
		Repo:          uuid.New(),
		SchedulerSpec: testhelpers.CompleteScheduler(),
	}
	leaves := []*pipelinescheduler.SchedulerLeaf{
		leaf1,
		leaf2,
	}
	cfg, _, err := pipelinescheduler.BuildProwConfig(leaves)
	assert.NoError(t, err)
	assert.Len(t, cfg.Postsubmits, 2)
	assert.Len(t, cfg.Presubmits, 2)
	repoName := fmt.Sprintf("%s/%s", org, leaf1.Repo)
	assert.Len(t, cfg.Presubmits[repoName], 1, "for repo name %s", repoName)
	assert.Equal(t, leaf1.Presubmits.Items[0].Name, cfg.Presubmits[repoName][0].Name, "for repo name %s", repoName)
}

func TestRepo(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "repo"), "config.yaml", "",
		[]testhelpers.SchedulerFile{
			{
				Filenames: []string{"repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestMultipleContexts(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "multiple_contexts"), "config.yaml", "",
		[]testhelpers.SchedulerFile{
			{
				Filenames: []string{"repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestWithParent(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "with_parent"), "config.yaml",
		"plugins.yaml", []testhelpers.SchedulerFile{
			{
				Filenames: []string{"parent.yaml", "repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestNoPostSubmitsWithParent(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "no_postsubmits_with_parent"), "config.yaml",
		"plugins.yaml", []testhelpers.SchedulerFile{
			{
				Filenames: []string{"parent.yaml", "repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestPolicyWithParent(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "policy_with_parent"), "config.yaml",
		"plugins.yaml", []testhelpers.SchedulerFile{
			{
				Filenames: []string{"parent.yaml", "repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestMergerWithParent(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "merger_with_parent"), "config.yaml",
		"plugins.yaml", []testhelpers.SchedulerFile{
			{
				Filenames: []string{"parent.yaml", "repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestMergerWithMergeMethod(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "merger_with_mergemethod"), "config.yaml",
		"plugins.yaml", []testhelpers.SchedulerFile{
			{
				Filenames: []string{"parent.yaml", "repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestOnlyWithParent(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "only_with_parent"), "config.yaml",
		"plugins.yaml", []testhelpers.SchedulerFile{
			{
				Filenames: []string{"parent.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestOnlyPluginsFromRepo(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "only_plugins_from_repo"), "",
		"plugins.yaml", []testhelpers.SchedulerFile{
			{
				Filenames: []string{"parent.yaml", "repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestOnlyPluginsJustFromParent(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "only_plugins_from_parent"), "",
		"plugins.yaml", []testhelpers.SchedulerFile{
			{
				Filenames: []string{"parent.yaml", "repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}

func TestOnlyPluginsMixFromParentAndRepo(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	testhelpers.BuildAndValidateProwConfig(t, filepath.Join(wd, "testdata", "only_plugins"), "",
		"plugins.yaml", []testhelpers.SchedulerFile{
			{
				Filenames: []string{"parent.yaml", "repo.yaml"},
				Org:       "acme",
				Repo:      "dummy",
			},
		})
}
