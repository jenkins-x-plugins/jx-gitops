package status

import (
	"context"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/releasereport"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"path/filepath"
	"testing"
)

const (
	prod    = "prod"
	preProd = "pre-prod"
	staging = "staging"
	preview = "preview"

	repoOwner = "fakeOwner"
	repoName  = "fakeRepo"
)

var fullRepoName = filepath.Join(repoOwner, repoName)

func TestHemlfileStatus(t *testing.T) {
	_, o := NewCmdHelmfileStatus()
	o.Dir = "testdata"
	o.TestGitToken = "faketoken"
	err := o.Run()
	require.NoError(t, err, "failed to run")
}

func TestNewCmdHelmfileStatus_FindExistingDeployment(t *testing.T) {
	fakeDeployments := []*scm.Deployment{
		{
			Name:        repoName,
			Environment: prod,
		},
		{
			Name:        repoName,
			Environment: preProd,
		},
		{
			Name:        repoName,
			Environment: staging,
		},
	}

	type inputArgs struct {
		fullRepoName   string
		deploymentName string
		environment    string
	}

	testCases := []struct {
		name               string
		testArgs           inputArgs
		currentDeployments []*scm.Deployment
		expectedDeployment *scm.Deployment
	}{
		{
			name: "correct name and env",
			testArgs: inputArgs{
				fullRepoName: fullRepoName,
				environment:  prod,
			},
			expectedDeployment: &scm.Deployment{
				Name:        repoName,
				Environment: prod,
			},
		},
		{
			name: "no existing deployment for name",
			testArgs: inputArgs{
				fullRepoName: repoOwner + "/reallyFakeRepo",
				environment:  prod,
			},
			expectedDeployment: nil,
		},
		{
			name: "no existing deployment for env",
			testArgs: inputArgs{
				fullRepoName: fullRepoName,
				environment:  preview,
			},
			expectedDeployment: nil,
		},
	}

	testOpts := Options{}

	var data *fake.Data
	testOpts.ScmClient, data = fake.NewDefault()
	data.Deployments = map[string][]*scm.Deployment{fullRepoName: fakeDeployments}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			actualDeployment, err := testOpts.FindExistingDeploymentInEnvironment(context.TODO(), tt.testArgs.fullRepoName, tt.testArgs.environment)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedDeployment, actualDeployment)
		})
	}
}

func TestNewCmdHelmfileStatus_UpdateStatus(t *testing.T) {
	type inputArgs struct {
		env     *environment
		repo    *v1alpha1.Repository
		group   *v1alpha1.RepositoryGroup
		release *releasereport.ReleaseInfo
	}

	testCases := []struct {
		name                string
		inputArgs           inputArgs
		initialDeployments  []*scm.Deployment
		expectedDeployments []*scm.Deployment
		expectedStatuses    map[string][]*scm.DeploymentStatus
	}{
		{
			name: "existing deployment (create new status)",
			inputArgs: inputArgs{
				env:     &environment{name: prod},
				repo:    &v1alpha1.Repository{Name: repoName},
				group:   &v1alpha1.RepositoryGroup{Owner: repoOwner},
				release: &releasereport.ReleaseInfo{Metadata: chart.Metadata{Version: "1"}},
			},
			initialDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: prod,
					ID:          "deployment-1",
				},
			},
			expectedDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: prod,
					ID:          "deployment-1",
				},
			},
			expectedStatuses: map[string][]*scm.DeploymentStatus{
				fullRepoName + "/deployment-1": {
					{
						ID:          "status-1",
						State:       "success",
						Description: "Deployment 1",
						Environment: prod,
					},
				},
			},
		},
		{
			name: "no existing deployment (create new deployment & status)",
			inputArgs: inputArgs{
				env:     &environment{name: prod},
				repo:    &v1alpha1.Repository{Name: repoName},
				group:   &v1alpha1.RepositoryGroup{Owner: repoOwner},
				release: &releasereport.ReleaseInfo{Metadata: chart.Metadata{Version: "1"}},
			},
			initialDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: preProd,
					ID:          "deployment-1",
				},
			},
			expectedDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: preProd,
					ID:          "deployment-1",
				},
				{
					Name:                  repoName,
					Environment:           prod,
					ID:                    "deployment-2",
					Namespace:             repoOwner,
					Ref:                   "v1",
					Task:                  "deploy",
					Description:           "release fakeRepo for version 1",
					ProductionEnvironment: true,
					OriginalEnvironment:   prod,
					Payload:               "",
				},
			},
			expectedStatuses: map[string][]*scm.DeploymentStatus{
				fullRepoName + "/deployment-2": {
					{
						ID:          "status-1",
						State:       "success",
						Description: "Deployment 1",
						Environment: prod,
					},
				},
			},
		},
		{
			name: "existing deployment with same ref (skip deployment & status)",
			inputArgs: inputArgs{
				env:     &environment{name: prod},
				repo:    &v1alpha1.Repository{Name: repoName},
				group:   &v1alpha1.RepositoryGroup{Owner: repoOwner},
				release: &releasereport.ReleaseInfo{Metadata: chart.Metadata{Version: "1"}},
			},
			initialDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: prod,
					ID:          "deployment-1",
					Ref:         "v1",
				},
			},
			expectedDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: prod,
					ID:          "deployment-1",
					Ref:         "v1",
				},
			},
			expectedStatuses: map[string][]*scm.DeploymentStatus{},
		},
		{
			name: "no release version (skip deployment & status)",
			inputArgs: inputArgs{
				env:     &environment{name: prod},
				repo:    &v1alpha1.Repository{Name: repoName},
				group:   &v1alpha1.RepositoryGroup{Owner: repoOwner},
				release: &releasereport.ReleaseInfo{Metadata: chart.Metadata{Version: ""}},
			},
			initialDeployments:  []*scm.Deployment{},
			expectedDeployments: []*scm.Deployment{},
			expectedStatuses:    map[string][]*scm.DeploymentStatus{},
		},
	}

	for _, tt := range testCases {
		testOpts := Options{}

		var data *fake.Data
		testOpts.ScmClient, data = fake.NewDefault()

		data.Deployments[fullRepoName] = tt.initialDeployments

		t.Run(tt.name, func(t *testing.T) {
			err := testOpts.updateStatus(context.TODO(), tt.inputArgs.env, tt.inputArgs.repo, tt.inputArgs.group, tt.inputArgs.release)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatuses, data.DeploymentStatus)
			assert.Equal(t, tt.expectedDeployments, data.Deployments[fullRepoName])
		})
	}
}
