package v1alpha1

import (
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SourceConfigFileName default name of the source repository configuration
	SourceConfigFileName = "source-config.yaml"

	//DefaultSlackChannel
	DefaultSlackChannel = "#jenkins-x-pipelines"
)

// BooleanFlag a type that is used for string boolean values that can be blank or yes/no
type BooleanFlag string

var (
	// BooleanFlagNone indicates no value
	BooleanFlagNone BooleanFlag = ""

	// BooleanFlagYes BooleanFlagYes indicates yes
	BooleanFlagYes BooleanFlag = "yes"

	// BooleanFlagNo indicates no
	BooleanFlagNo BooleanFlag = "no"
)

// PipelineKind what pipeline to notify on
type PipelineKind string

var (
	// PipelineKindNone indicates all pipelines
	PipelineKindNone PipelineKind = ""

	// PipelineKindAll indicates all pipelines
	PipelineKindAll PipelineKind = "all"

	// PipelineKindRelease only notify on release pipelines
	PipelineKindRelease PipelineKind = "release"

	// PipelineKindPullRequest only notify on pullRequest pipelines
	PipelineKindPullRequest PipelineKind = "pullRequest"
)

// NotifyKind what kind of notification
type NotifyKind string

var (
	// NotifyKindNone indicates no notification
	NotifyKindNone NotifyKind = ""

	// NotifyKindNever never notify
	NotifyKindNever NotifyKind = "never"

	// NotifyKindAlways always notify
	NotifyKindAlways NotifyKind = "always"

	// NotifyKindFailure only failures
	NotifyKindFailure NotifyKind = "failure"

	// NotifyKindFailureOrFirstSuccess only failures or first success after failure
	NotifyKindFailureOrFirstSuccess NotifyKind = "failureOrNextSuccess"

	// NotifyKindSuccess only successful
	NotifyKindSuccess NotifyKind = "success"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SourceConfig represents a collection source repostory groups and repositories
//
// +k8s:openapi-gen=true
type SourceConfig struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the SourceConfig from the client
	// +optional
	Spec SourceConfigSpec `json:"spec"`
}

// SourceConfigList contains a list of SourceConfig
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SourceConfigList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SourceConfig `json:"items"`
}

// SourceConfigSpec defines the desired state of SourceConfig.
type SourceConfigSpec struct {
	// Groups the groups of source repositories
	Groups []RepositoryGroup `json:"groups,omitempty"`

	// Scheduler the default scheduler for any group/repository which does not specify one
	Scheduler string `json:"scheduler,omitempty"`

	// Slack optional default slack notification configuration inherited by groups
	Slack *SlackNotify `json:"slack,omitempty"`

	// JenkinsServers the jenkins servers configured for this repository
	JenkinsServers []JenkinsServer `json:"jenkinsServers,omitempty"`

	// JenkinsFolderTemplate the default template file to use to generate the folder job DSL script
	JenkinsFolderTemplate string `json:"jenkinsFolderTemplate,omitempty"`

	// JenkinsJobTemplate the default template file to use to generate the projects job DSL script
	JenkinsJobTemplate string `json:"jenkinsJobTemplate,omitempty"`
}

// SourceConfigSpec defines the desired state of SourceConfig.
type RepositoryGroup struct {
	// Provider the git provider server URL
	Provider string `json:"provider,omitempty"`

	// ProviderKind the git provider kind
	ProviderKind string `json:"providerKind,omitempty"`

	// ProviderName the git provider name
	ProviderName string `json:"providerName,omitempty"`

	// Owner the name of the organisation/owner/project/user that owns the repository
	Owner string `json:"owner,omitempty" validate:"nonzero"`

	// Repositories the repositories for the
	Repositories []Repository `json:"repositories,omitempty"`

	// Scheduler the default scheduler for this group
	Scheduler string `json:"scheduler,omitempty"`

	// JenkinsFolderTemplate the default template file to use to generate the folder job DSL script
	JenkinsFolderTemplate string `json:"jenkinsFolderTemplate,omitempty"`

	// JenkinsJobTemplate the default job template file to use to generate the projects job DSL script
	JenkinsJobTemplate string `json:"jenkinsJobTemplate,omitempty"`

	// Slack optional slack notification configuration
	Slack *SlackNotify `json:"slack,omitempty"`
}

// Repository the name of the repository to import and the optional scheduler
type Repository struct {
	// Name the name of the repository
	Name string `json:"name,omitempty" validate:"nonzero"`

	// Scheduler the optional name of the scheduler to use if different to the group
	Scheduler string `json:"scheduler,omitempty"`

	// JenkinsJobTemplate the template file to use to generate the projects job DSL script
	JenkinsJobTemplate string `json:"jenkinsJobTemplate,omitempty"`

	// Description the optional description of this repository
	Description string `json:"description,omitempty"`

	// URL the URL to access this repository
	URL string `json:"url,omitempty"`

	// HTTPCloneURL the HTTP/HTTPS based clone URL
	HTTPCloneURL string `json:"httpCloneURL,omitempty"`

	// SSHCloneURL the SSH based clone URL
	SSHCloneURL string `json:"sshCloneURL,omitempty"`

	// Slack optional slack notification configuration
	Slack *SlackNotify `json:"slack,omitempty"`
}

// JenkinsServer the Jenkins server configuration
type JenkinsServer struct {
	// Server the name of the Jenkins Server to use
	Server string `json:"server,omitempty"`

	// FolderTemplate the default template file to use to generate the folder job DSL script
	FolderTemplate string `json:"folderTemplate,omitempty"`

	// JobTemplate the default template file to use to generate the projects job DSL script
	JobTemplate string `json:"jobTemplate,omitempty"`

	// Groups the groups of source repositories
	Groups []RepositoryGroup `json:"groups,omitempty"`
}

// SlackNotify the slack notification configuration
type SlackNotify struct {
	// Channel the name of the channel to notify pipelines
	Channel string `json:"channel,omitempty"`

	// Kind kind of notification
	Kind NotifyKind `json:"kind,omitempty"`

	// Pipeline kind of pipeline to notify on
	Pipeline PipelineKind `json:"pipeline,omitempty"`

	// DirectMessage whether to use Direct Messages
	DirectMessage BooleanFlag `json:"directMessage,omitempty"`

	// NotifyReviewers whether to use Direct Messages
	NotifyReviewers BooleanFlag `json:"noDirectMessage,omitempty"`

	// Branch specify the branch name or filter to notify
	Branch *Pattern `json:"branch,omitempty"`

	// Context specify the context name or filter to notify
	Context *Pattern `json:"context,omitempty"`

	// PullRequestLabel specify the label pull request label to notify
	PullRequestLabel *Pattern `json:"pullRequestLabel,omitempty"`
}

// Pattern for matching strings
type Pattern struct {
	// Name
	Name string `json:"name,omitempty"`
	// Includes patterns to include in changing
	Includes []string `json:"include,omitempty"`
	// Excludes patterns to exclude from upgrading
	Excludes []string `json:"exclude,omitempty"`
}

// Matches returns true if the text matches the given text
func (p *Pattern) Matches(text string) bool {
	if p == nil {
		return true
	}
	if p.Name != "" {
		return text == p.Name
	}
	return stringhelpers.StringMatchesAny(text, p.Includes, p.Excludes)
}

// Matches returns true if the text matches the given text
func (p *Pattern) MatchesLabels(labels []string) bool {
	if p == nil {
		return true
	}
	if p.Name != "" {
		if stringhelpers.StringArrayIndex(labels, p.Name) < 0 {
			return false
		}
	}
	for _, text := range p.Excludes {
		if stringhelpers.StringArrayIndex(labels, text) >= 0 {
			return false
		}
	}
	if len(p.Includes) == 0 {
		return true
	}
	for _, text := range p.Includes {
		if stringhelpers.StringArrayIndex(labels, text) >= 0 {
			return true
		}
	}
	return false
}

func (p *Pattern) Inherit(group *Pattern) *Pattern {
	if p == nil {
		return group
	}
	return p
}

// ToBool converts the flag to a boolean such that it is only true if the
// value is "yes"
func (f BooleanFlag) ToBool() bool {
	return strings.ToLower(string(f)) == "yes"
}

// Inherit if the current flag is blank lets use the group value
func (f BooleanFlag) Inherit(group BooleanFlag) BooleanFlag {
	if string(f) != "" {
		return f
	}
	return group
}

// Inherit inherits the settings on this repository from the group
func (repo *SlackNotify) Inherit(group *SlackNotify) *SlackNotify {
	if repo == nil {
		return group
	}
	if group == nil {
		return repo
	}
	answer := *repo
	if repo.Channel == "" {
		answer.Channel = group.Channel
	}
	if string(repo.Pipeline) == "" {
		answer.Pipeline = group.Pipeline
	}
	if string(repo.Kind) != "" {
		answer.Kind = repo.Kind
	}

	answer.Branch = repo.Branch.Inherit(group.Branch)
	answer.Context = repo.Context.Inherit(group.Context)
	answer.PullRequestLabel = repo.PullRequestLabel.Inherit(group.PullRequestLabel)
	answer.DirectMessage = repo.DirectMessage.Inherit(group.DirectMessage)
	answer.NotifyReviewers = repo.NotifyReviewers.Inherit(group.NotifyReviewers)
	return &answer
}
