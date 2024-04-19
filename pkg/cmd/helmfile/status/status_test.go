package status

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/releasereport"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
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

func TestHemlfileStatus_WithEarlyCutoff(t *testing.T) {
	_, o := NewCmdHelmfileStatus()
	o.Dir = "testdata"
	o.TestGitToken = "faketoken"
	o.DeployOffset = ""
	cutoff, err := time.Parse(time.RFC3339, "2023-01-25T08:38:47Z")
	require.NoError(t, err, "failed to parse time")
	o.DeployCutoff = cutoff
	err = o.Run()
	require.NoError(t, err, "failed to run")
	testFullRepoName := filepath.Join("jstrachan", "nodey560")
	deployments, _, _ := o.ScmClient.Deployments.List(context.Background(), testFullRepoName, &scm.ListOptions{})
	require.Len(t, deployments, 1)
	require.Equal(t, "Production", deployments[0].Environment)
}

func TestHemlfileStatus_WithLateCutoff(t *testing.T) {
	_, o := NewCmdHelmfileStatus()
	o.Dir = "testdata"
	o.TestGitToken = "faketoken"
	o.DeployOffset = ""
	cutoff, err := time.Parse(time.RFC3339, "2023-01-25T06:38:47Z")
	require.NoError(t, err, "failed to parse time")
	o.DeployCutoff = cutoff
	err = o.Run()
	require.NoError(t, err, "failed to run")
	testFullRepoName := filepath.Join("jstrachan", "nodey560")
	deployments, _, _ := o.ScmClient.Deployments.List(context.Background(), testFullRepoName, &scm.ListOptions{})
	require.Len(t, deployments, 2)
	require.Equal(t, "Production", deployments[0].Environment)
	require.Equal(t, "Staging", deployments[1].Environment)
}

func TestNewCmdHelmfileStatus_FindExistingDeployment(t *testing.T) {
	fakeDeployments := []*scm.Deployment{
		{
			Name:        repoName,
			Environment: prod,
			Ref:         "v1.0.0",
		},
		{
			Name:        repoName,
			Environment: preProd,
			Ref:         "v1.5.0",
		},
		{
			Name:        repoName,
			Environment: staging,
			Ref:         "v2.0.1",
		},
	}

	type inputArgs struct {
		fullRepoName string
		environment  string
		ref          string
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
				ref:          "v1.0.0",
			},
			expectedDeployment: &scm.Deployment{
				Name:        repoName,
				Environment: prod,
				Ref:         "v1.0.0",
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
			actualDeployment, err := testOpts.FindExistingDeploymentInEnvironment(context.TODO(), tt.testArgs.ref, tt.testArgs.environment, tt.testArgs.fullRepoName)
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
			name: "add deployment (other ref exists)",
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
					Ref:         "v0",
				},
			},
			expectedDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: prod,
					ID:          "deployment-1",
					Ref:         "v0",
				},
				{
					ID:                    "deployment-2",
					Namespace:             repoOwner,
					Name:                  repoName,
					Ref:                   "v1",
					Task:                  "deploy",
					Description:           "release fakeRepo for reference 1",
					OriginalEnvironment:   "prod",
					Environment:           prod,
					ProductionEnvironment: true,
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
					Ref:         "v0",
				},
			},
			expectedDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: preProd,
					ID:          "deployment-1",
					Ref:         "v0",
				},
				{
					Name:                  repoName,
					Environment:           prod,
					ID:                    "deployment-2",
					Namespace:             repoOwner,
					Ref:                   "v1",
					Task:                  "deploy",
					Description:           "release fakeRepo for reference 1",
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
		{
			name: "non standard git reference with no existing deployment (create new deployment & status)",
			inputArgs: inputArgs{
				env:     &environment{name: prod},
				repo:    &v1alpha1.Repository{Name: repoName},
				group:   &v1alpha1.RepositoryGroup{Owner: repoOwner},
				release: &releasereport.ReleaseInfo{Metadata: chart.Metadata{Version: "1", Annotations: map[string]string{"gitReference": "fafe062ebf497187b3ce7b47573580b4330f78b4924fb38e4c1e8db128711720"}}},
			},
			initialDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: preProd,
					ID:          "deployment-1",
					Ref:         "e8cc95b323e85788dc82398cd37a259aff1a0bf2ca5489d5af4201aa4eca3743",
				},
			},
			expectedDeployments: []*scm.Deployment{
				{
					Name:        repoName,
					Environment: preProd,
					ID:          "deployment-1",
					Ref:         "e8cc95b323e85788dc82398cd37a259aff1a0bf2ca5489d5af4201aa4eca3743",
				},
				{
					Name:                  repoName,
					Environment:           prod,
					ID:                    "deployment-2",
					Namespace:             repoOwner,
					Ref:                   "fafe062ebf497187b3ce7b47573580b4330f78b4924fb38e4c1e8db128711720",
					Task:                  "deploy",
					Description:           "release fakeRepo for reference fafe062ebf497187b3ce7b47573580b4330f78b4924fb38e4c1e8db128711720",
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
	}

	for _, tt := range testCases {
		testOpts := Options{}

		var data *fake.Data
		testOpts.ScmClient, data = fake.NewDefault()

		data.Deployments[fullRepoName] = tt.initialDeployments

		t.Run(tt.name, func(t *testing.T) {
			err := testOpts.updateStatus(context.TODO(), tt.inputArgs.env, tt.inputArgs.group.Provider, tt.inputArgs.group.Owner, tt.inputArgs.repo.Name, tt.inputArgs.release)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatuses, data.DeploymentStatus)
			assert.Equal(t, tt.expectedDeployments, data.Deployments[fullRepoName])
		})
	}
}
