package sourceconfigs_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	gitKind   = "github"
	gitServer = giturl.GitHubURL
)

func TestSourceConfigDefaultValues(t *testing.T) {
	owner := "myowner"

	config := &v1alpha1.SourceConfig{
		Spec: v1alpha1.SourceConfigSpec{
			Groups: []v1alpha1.RepositoryGroup{
				{
					Provider:     gitServer,
					ProviderKind: gitKind,
					Owner:        owner,
					Repositories: []v1alpha1.Repository{
						{
							Name: "no-cfg",
						},
						{
							Name: "override-channel",
							Slack: &v1alpha1.SlackNotify{
								Channel:  "new-channel",
								Pipeline: v1alpha1.PipelineKindAll,
							},
						},
					},
					Slack: &v1alpha1.SlackNotify{
						Channel: "default-channel",
					},
				},
				{
					Provider:     gitServer,
					ProviderKind: gitKind,
					Owner:        "default-disabled",
					Repositories: []v1alpha1.Repository{
						{
							Name: "default-value",
						},
						{
							Name: "repo-enabled",
							Slack: &v1alpha1.SlackNotify{
								DirectMessage: v1alpha1.BooleanFlagNo,
							},
						},
					},
					Slack: &v1alpha1.SlackNotify{
						Channel:       "default-channel",
						DirectMessage: v1alpha1.BooleanFlagYes,
					},
				},
				{
					Provider:     gitServer,
					ProviderKind: gitKind,
					Owner:        "no-cfg",
					Repositories: []v1alpha1.Repository{
						{
							Name: "default-value",
						},
						{
							Name: "repo-enabled",
							Slack: &v1alpha1.SlackNotify{
								DirectMessage:   v1alpha1.BooleanFlagNo,
								NotifyReviewers: v1alpha1.BooleanFlagNo,
							},
						},
					},
				},
			},
			Slack: &v1alpha1.SlackNotify{
				Channel:         "default-channel-for-orgs",
				Pipeline:        v1alpha1.PipelineKindRelease,
				NotifyReviewers: v1alpha1.BooleanFlagYes,
			},
		},
	}

	err := sourceconfigs.DefaultConfigValues(config)
	require.NoError(t, err)

	assertSlackChannel(t, config, owner, "no-cfg", "default-channel", v1alpha1.PipelineKindRelease, false, true, true)
	assertSlackChannel(t, config, owner, "override-channel", "new-channel", v1alpha1.PipelineKindAll, false, true, true)

	assertSlackChannel(t, config, "default-disabled", "default-value", "default-channel", v1alpha1.PipelineKindRelease, true, true, true)
	assertSlackChannel(t, config, "default-disabled", "repo-enabled", "default-channel", v1alpha1.PipelineKindRelease, false, true, true)

	assertSlackChannel(t, config, "no-cfg", "default-value", "default-channel-for-orgs", v1alpha1.PipelineKindRelease, false, true, true)
	assertSlackChannel(t, config, "no-cfg", "repo-enabled", "default-channel-for-orgs", v1alpha1.PipelineKindRelease, false, false, true)
}

func TestSourceConfigGlobalDefaultValues(t *testing.T) {
	owner := "myowner"

	nonGroupConfig := &v1alpha1.SourceConfig{
		Spec: v1alpha1.SourceConfigSpec{
			Groups: []v1alpha1.RepositoryGroup{},
			Slack: &v1alpha1.SlackNotify{
				Channel:  "my-channel",
				Pipeline: v1alpha1.PipelineKindAll,
			},
		},
	}

	config := &v1alpha1.SourceConfig{
		Spec: v1alpha1.SourceConfigSpec{
			Groups: []v1alpha1.RepositoryGroup{
				{
					Provider:     gitServer,
					ProviderKind: gitKind,
					Owner:        owner,
					Repositories: []v1alpha1.Repository{
						{
							Name: "myrepo",
						},
					},
				},
				{
					Provider:     gitServer,
					ProviderKind: gitKind,
					Owner:        owner,
					Repositories: []v1alpha1.Repository{
						{
							Name: "myrepo2",
						},
					},
				},
			},
			Slack: &v1alpha1.SlackNotify{
				Channel:  "my-channel",
				Pipeline: v1alpha1.PipelineKindAll,
			},
		},
	}

	err := sourceconfigs.DefaultConfigValues(config)
	require.NoError(t, err)

	assertSlackChannel(t, nonGroupConfig, "", "", "", v1alpha1.PipelineKindAll, false, false, false)
	assertSlackChannel(t, config, owner, "myrepo2", "my-channel", v1alpha1.PipelineKindAll, false, false, true)
}

func assertSlackChannel(t *testing.T, config *v1alpha1.SourceConfig, owner, repoName, expectedChannel string, expectedPipeline v1alpha1.PipelineKind, expectedDirectMessage, expectedNotifyReviewers, expectedRepo bool) {
	repo := sourceconfigs.GetRepositoryFor(config, "", owner, repoName)
	if expectedRepo {
		require.NotNil(t, repo, "should have found a repo for owner %s and repo %s", owner, repoName)
		slack := repo.Slack
		require.NotNil(t, slack, "no slack configuration found for owner %s and repo %s", owner, repoName)
		assert.Equal(t, expectedChannel, slack.Channel, "slack channel for owner %s and repo %s", owner, repoName)
		assert.Equal(t, expectedPipeline, slack.Pipeline, "slack pipeline for owner %s and repo %s", owner, repoName)
		assert.Equal(t, expectedDirectMessage, slack.DirectMessage.ToBool(), "slack channel directMessage flag for owner %s and repo %s", owner, repoName)
		assert.Equal(t, expectedNotifyReviewers, slack.NotifyReviewers.ToBool(), "slack channel notifyReviewers flag for owner %s and repo %s", owner, repoName)
	} else {
		require.Nil(t, repo, "should not have found a repo for owner %s and repo %s", owner, repoName)
	}
}
