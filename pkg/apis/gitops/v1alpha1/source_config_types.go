package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// SourceConfigFileName default name of the source repository configuration
	SourceConfigFileName = "source-config.yaml"
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

	// JenkinsServers the jenkins servers configured for this repository
	JenkinsServers []JenkinsServer `json:"jenkinsServers,omitempty"`

	// JenkinsJobTemplate the default configuration template file to use to generate the projects job DSL script
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

	// JenkinsJobTemplate the default configuration template file to use to generate the projects job DSL script
	JenkinsJobTemplate string `json:"jenkinsJobTemplate,omitempty"`
}

// Repository the name of the repository to import and the optional scheduler
type Repository struct {
	// Name the name of the repository
	Name string `json:"name,omitempty" validate:"nonzero"`

	// Scheduler the optional name of the scheduler to use if different to the group
	Scheduler string `json:"scheduler,omitempty"`

	// JenkinsJobTemplate the configuration template file to use to generate the projects job DSL script
	JenkinsJobTemplate string `json:"jenkinsJobTemplate,omitempty"`

	// Description the optional description of this repository
	Description string `json:"description,omitempty"`

	// URL the URL to access this repository
	URL string `json:"url,omitempty"`

	// HTTPCloneURL the HTTP/HTTPS based clone URL
	HTTPCloneURL string `json:"httpCloneURL,omitempty"`

	// SSHCloneURL the SSH based clone URL
	SSHCloneURL string `json:"sshCloneURL,omitempty"`
}

// JenkinsServer the Jenkins server configuration
type JenkinsServer struct {
	// Server the name of the Jenkins Server to use
	Server string `json:"server,omitempty"`

	// JobTemplate the default configuration template file to use to generate the projects job DSL script
	JobTemplate string `json:"jobTemplate,omitempty"`

	// Groups the groups of source repositories
	Groups []RepositoryGroup `json:"groups,omitempty"`
}
