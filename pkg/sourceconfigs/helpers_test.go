package sourceconfigs_test

import (
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-gitops/pkg/sourceconfigs"
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
								Channel: "new-channel",
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
				NotifyReviewers: v1alpha1.BooleanFlagYes,
			},
		},
	}

	err := sourceconfigs.DefaultConfigValues(config)
	require.NoError(t, err, "default values")

	assertSlackChannel(t, config, owner, "no-cfg", "default-channel", false, true)
	assertSlackChannel(t, config, owner, "override-channel", "new-channel", false, true)

	assertSlackChannel(t, config, "default-disabled", "default-value", "default-channel", true, true)
	assertSlackChannel(t, config, "default-disabled", "repo-enabled", "default-channel", false, true)

	assertSlackChannel(t, config, "no-cfg", "default-value", "default-channel-for-orgs", false, true)
	assertSlackChannel(t, config, "no-cfg", "repo-enabled", "default-channel-for-orgs", false, false)
}

func assertSlackChannel(t *testing.T, config *v1alpha1.SourceConfig, owner string, repoName string, expectedChannel string, expectedDirectMessage bool, expectedNotifyReviewers bool) {
	group := sourceconfigs.GetOrCreateGroup(config, gitKind, gitServer, owner)
	repo := sourceconfigs.GetOrCreateRepository(group, repoName)
	require.NotNil(t, repo, "should have found a repo for owner %s and repo %s", owner, repoName)
	slack := repo.Slack
	require.NotNil(t, slack, "no slack configuration found for owner %s and repo %s", owner, repoName)
	assert.Equal(t, expectedChannel, slack.Channel, "slack channel for owner %s and repo %s", owner, repoName)
	assert.Equal(t, expectedDirectMessage, slack.DirectMessage.ToBool(), "slack channel directMessage flag for owner %s and repo %s", owner, repoName)
	assert.Equal(t, expectedNotifyReviewers, slack.NotifyReviewers.ToBool(), "slack channel notifyReviewers flag for owner %s and repo %s", owner, repoName)
}
