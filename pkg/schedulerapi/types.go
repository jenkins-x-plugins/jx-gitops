package schedulerapi

import (
	"time"

	"github.com/jenkins-x/lighthouse/pkg/config/job"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:openapi-gen=true

// Scheduler is configuration for a pipeline scheduler
type Scheduler struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	Spec SchedulerSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SchedulerList is a list of configurations for a pipeline scheduler
type SchedulerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Scheduler `json:"items"`
}

// TODO Support Label plugin?
// TODO Support Size plugin?
// TODO Support Welcome plugin?
// TODO Support Blockade plugin?
// TODO Support Golint plugin?
// TODO Support RepoMilestone plugin?
// TODO Support RequireMatchingLabel plugin?
// TODO Support Blunderbuss plugin?
// TODO Support Config Updater Plugin?
// TODO Support Owners plugin?
// TODO Support Heart plugin?
// TODO Support requiresig plugin?
// TODO support sigmention plugin?
// TODO support slack plugin?

// SchedulerSpec defines the pipeline scheduler (e.g. Prow) configuration
type SchedulerSpec struct {
	ScehdulerAgent  *SchedulerAgent                    `json:"schedulerAgent,omitempty" protobuf:"bytes,1,opt,name=schedulerAgent"`
	Policy          *GlobalProtectionPolicy            `json:"policy,omitempty" protobuf:"bytes,2,opt,name=policy"`
	Presubmits      *Presubmits                        `json:"presubmits,omitempty" protobuf:"bytes,3,opt,name=presubmits"`
	Postsubmits     *Postsubmits                       `json:"postsubmits,omitempty" protobuf:"bytes,4,opt,name=postsubmits"`
	Trigger         *Trigger                           `json:"trigger,omitempty" protobuf:"bytes,5,opt,name=trigger"`
	Approve         *Approve                           `json:"approve,omitempty" protobuf:"bytes,6,opt,name=approve"`
	LGTM            *Lgtm                              `json:"lgtm,omitempty" protobuf:"bytes,7,opt,name=lgtm"`
	ExternalPlugins *ReplaceableSliceOfExternalPlugins `json:"external_plugins,omitempty" protobuf:"bytes,8,opt,name=external_plugins"`

	Merger *Merger `json:"merger,omitempty" protobuf:"bytes,9,opt,name=merger"`

	// Plugins is a list of plugin names enabled for a repo
	Plugins       *ReplaceableSliceOfStrings `json:"plugins,omitempty" protobuf:"bytes,10,opt,name=plugins"`
	ConfigUpdater *ConfigUpdater             `json:"config_updater,omitempty" protobuf:"bytes,11,opt,name=config_updater"`
	Welcome       []*Welcome                 `json:"welcome,omitempty" protobuf:"bytes,12,opt,name=welcome"`
	Periodics     *Periodics                 `json:"periodics,omitempty" protobuf:"bytes,13,opt,name=periodics"`
	Attachments   []*Attachment              `json:"attachments,omitempty" protobuf:"bytes,13,opt,name=attachments"`
}

// ConfigMapSpec contains configuration options for the configMap being updated
// by the config-updater plugin.
type ConfigMapSpec struct {
	// Name of ConfigMap
	Name string `json:"name"`
	// Key is the key in the ConfigMap to update with the file contents.
	// If no explicit key is given, the basename of the file will be used.
	Key string `json:"key,omitempty"`
	// Namespace in which the configMap needs to be deployed. If no namespace is specified
	// it will be deployed to the ProwJobNamespace.
	Namespace string `json:"namespace,omitempty"`
	// Namespaces in which the configMap needs to be deployed, in addition to the above
	// namespace provided, or the default if it is not set.
	AdditionalNamespaces []string `json:"additional_namespaces,omitempty"`

	// Namespaces is the fully resolved list of Namespaces to deploy the ConfigMap in
	Namespaces []string `json:"-"`
}

// ConfigUpdater holds configuration for the config updater plugin
type ConfigUpdater struct {
	Map        map[string]ConfigMapSpec `json:"map,omitempty" protobuf:"bytes,1,opt,name=map"`
	ConfigFile string                   `json:"config_file,omitempty" protobuf:"bytes,2,opt,name=config_file"`
	PluginFile string                   `json:"plugin_file,omitempty" protobuf:"bytes,3,opt,name=plugin_file"`
	// +optional
	ConfigMap ConfigMapSpec
}

// ExternalPlugin holds configuration for registering an external
// plugin.
type ExternalPlugin struct {
	// Name of the plugin.
	Name *string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// Endpoint is the location of the external plugin. Defaults to
	// the name of the plugin, ie. "http://{{name}}".
	Endpoint *string `json:"endpoint,omitempty" protobuf:"bytes,2,opt,name=endpoint"`
	// ReplaceableSliceOfStrings are the events that need to be demuxed by the hook
	// server to the external plugin. If no events are specified,
	// everything is sent.
	Events *ReplaceableSliceOfStrings `json:"events,omitempty" protobuf:"bytes,3,opt,name=events"`
}

// ReplaceableSliceOfStrings is a slice of strings that can optionally completely replace the slice of strings
// defined in the parent scheduler
type ReplaceableSliceOfStrings struct {
	// Items is the string values
	Items []string `json:"entries,omitempty" protobuf:"bytes,1,opt,name=entries"`
	// Replace the existing entries
	Replace bool `json:"replace,omitempty" protobuf:"bytes,2,opt,name=replace"`
}

// Lgtm specifies a configuration for a single lgtm.
// The configuration for the lgtm plugin is defined as a list of these structures.
type Lgtm struct {
	// ReviewActsAsLgtm indicates that a Github review of "approve" or "request changes"
	// acts as adding or removing the lgtm label
	ReviewActsAsLgtm *bool `json:"review_acts_as_lgtm,omitempty" protobuf:"bytes,1,opt,name=review_acts_as_lgtm"`
	// StoreTreeHash indicates if tree_hash should be stored inside a comment to detect
	// squashed commits before removing lgtm labels
	StoreTreeHash *bool `json:"store_tree_hash,omitempty" protobuf:"bytes,2,opt,name=store_tree_hash"`
	// WARNING: This disables the security mechanism that prevents a malicious member (or
	// compromised GitHub account) from merging arbitrary code. Use with caution.
	//
	// StickyLgtmTeam specifies the Github team whose members are trusted with sticky LGTM,
	// which eliminates the need to re-lgtm minor fixes/updates.
	StickyLgtmTeam *string `json:"trusted_team_for_sticky_lgtm,omitempty" protobuf:"bytes,3,opt,name=stickyLgtmTeam"`
}

// Approve specifies a configuration for a single approve.
//
// The configuration for the approve plugin is defined as a list of these structures.
type Approve struct {
	// IssueRequired indicates if an associated issue is required for approval in
	// the specified repos.
	IssueRequired *bool `json:"issue_required,omitempty" protobuf:"bytes,1,opt,name=issue_required"`

	// RequireSelfApproval requires PR authors to explicitly approve their PRs.
	// Otherwise the plugin assumes the author of the PR approves the changes in the PR.
	RequireSelfApproval *bool `json:"require_self_approval,omitempty" protobuf:"bytes,2,opt,name=require_self_approval"`

	// LgtmActsAsApprove indicates that the lgtm command should be used to
	// indicate approval
	LgtmActsAsApprove *bool `json:"lgtm_acts_as_approve,omitempty" protobuf:"bytes,3,opt,name=lgtm_acts_as_approve"`

	// IgnoreReviewState causes the approve plugin to ignore the GitHub review state. Otherwise:
	// * an APPROVE github review is equivalent to leaving an "/approve" message.
	// * A REQUEST_CHANGES github review is equivalent to leaving an /approve cancel" message.
	IgnoreReviewState *bool `json:"ignore_review_state,omitempty" protobuf:"bytes,4,opt,name=ignore_review_state"`
}

// Trigger specifies a configuration for a single trigger.
//
// The configuration for the trigger plugin is defined as a list of these structures.
type Trigger struct {
	// TrustedOrg is the org whose members' PRs will be automatically built
	// for PRs to the above repos. The default is the PR's org.
	TrustedOrg *string `json:"trusted_org,omitempty" protobuf:"bytes,1,opt,name=trusted_org"`
	// JoinOrgURL is a link that redirects users to a location where they
	// should be able to read more about joining the organization in order
	// to become trusted members. Defaults to the Github link of TrustedOrg.
	JoinOrgURL *string `json:"join_org_url,omitempty" protobuf:"bytes,2,opt,name=join_org_url"`
	// OnlyOrgMembers requires PRs and/or /ok-to-test comments to come from org members.
	// By default, trigger also include repo collaborators.
	OnlyOrgMembers *bool `json:"only_org_members,omitempty" protobuf:"bytes,3,opt,name=only_org_members"`
	// IgnoreOkToTest makes trigger ignore /ok-to-test comments.
	// This is a security mitigation to only allow testing from trusted users.
	IgnoreOkToTest *bool `json:"ignore_ok_to_test,omitempty" protobuf:"bytes,4,opt,name=ignore_ok_to_test"`
	// ElideSkippedContexts makes trigger not post "Skipped" contexts for jobs
	// that could run but do not run.
	ElideSkippedContexts *bool `json:"elide_skipped_contexts,omitempty"`
}

// Postsubmits is a list of Postsubmit job configurations that can optionally completely replace the Postsubmit job
// configurations in the parent scheduler
type Postsubmits struct {
	// Items are the post submit configurations
	Items []*job.Postsubmit `json:"entries,omitempty" protobuf:"bytes,1,opt,name=entries"`
	// Replace the existing entries
	Replace bool `json:"replace,omitempty" protobuf:"bytes,2,opt,name=replace"`
}

// Presubmits is a list of Presubmit job configurations that can optionally completely replace the Presubmit job
// configurations in the parent scheduler
type Presubmits struct {
	// Items are the Presubmit configurtations
	Items []*Presubmit `json:"entries,omitempty" protobuf:"bytes,1,opt,name=entries"`
	// Replace the existing entries
	Replace bool `json:"replace,omitempty" protobuf:"bytes,2,opt,name=replace"`
}

type Presubmit struct {
	job.Presubmit

	// Override the default method of merge. Valid options are squash, rebase, and merge.
	MergeType *string `json:"merge_method,omitempty" protobuf:"bytes,7,opt,name=mergeMethod"`

	Queries []*Query `json:"queries,omitempty" protobuf:"bytes,8,opt,name=query"`

	Policy *ProtectionPolicies `json:"policy,omitempty" protobuf:"bytes,9,opt,name=policy"`
	// ContextOptions defines the merge options. If not set it will infer
	// the required and optional contexts from the jobs configured and use the Git Provider
	// combined status; otherwise it may apply the branch protection setting or let user
	// define their own options in case branch protection is not used.
	ContextPolicy *RepoContextPolicy `json:"context_options,omitempty" protobuf:"bytes,10,opt,name=contextPolicy"`
}

// Periodics is a list of jobs to be run periodically
type Periodics struct {
	// Items are the post submit configurations
	Items []*job.Periodic `json:"entries,omitempty" protobuf:"bytes,1,opt,name=entries"`
	// Replace the existing entries
	Replace bool `json:"replace,omitempty" protobuf:"bytes,2,opt,name=replace"`
}

// Query is turned into a Git Provider search query. See the docs for details:
// https://help.github.com/articles/searching-issues-and-pull-requests/
type Query struct {
	ExcludedBranches       *ReplaceableSliceOfStrings `json:"excludedBranches,omitempty" protobuf:"bytes,1,opt,name=excludedBranches"`
	IncludedBranches       *ReplaceableSliceOfStrings `json:"included_branches,omitempty" protobuf:"bytes,2,opt,name=included_branches"`
	Labels                 *ReplaceableSliceOfStrings `json:"labels,omitempty" protobuf:"bytes,3,opt,name=labels"`
	MissingLabels          *ReplaceableSliceOfStrings `json:"missingLabels,omitempty" protobuf:"bytes,4,opt,name=missingLabels"`
	Milestone              *string                    `json:"milestone,omitempty" protobuf:"bytes,5,opt,name=milestone"`
	ReviewApprovedRequired *bool                      `json:"review_approved_required,omitempty" protobuf:"bytes,6,opt,name=review_approved_required"`
}

// PullRequestMergeType enumerates the types of merges the Git Provider API can
// perform
// https://developer.github.com/v3/pulls/#merge-a-pull-request-merge-button
type PullRequestMergeType string

// Possible types of merges for the Git Provider merge API
const (
	MergeMerge  PullRequestMergeType = "merge"
	MergeRebase PullRequestMergeType = "rebase"
	MergeSquash PullRequestMergeType = "squash"
)

// Merger defines the options used to merge the PR
type Merger struct {
	// SyncPeriod specifies how often Merger will sync jobs with Github. Defaults to 1m.
	SyncPeriod *time.Duration `json:"-"`
	// StatusUpdatePeriod
	StatusUpdatePeriod *time.Duration `json:"-"`

	// URL for status contexts.
	TargetURL *string `json:"target_url,omitempty" protobuf:"bytes,1,opt,name=target_url"`

	// PRStatusBaseURL is the base URL for the PR status page.
	// This is used to link to a merge requirements overview
	// in the status context.
	PRStatusBaseURL *string `json:"pr_status_base_url,omitempty" protobuf:"bytes,2,opt,name=prStatusBaseURL"`

	// BlockerLabel is an optional label that is used to identify merge blocking
	// Git Provider issues.
	// Leave this blank to disable this feature and save 1 API token per sync loop.
	BlockerLabel *string `json:"blockerLabel,omitempty"`

	// SquashLabel is an optional label that is used to identify PRs that should
	// always be squash merged.
	// Leave this blank to disable this feature.
	SquashLabel *string `json:"squashLabel,omitempty"`

	// MaxGoroutines is the maximum number of goroutines spawned inside the
	// controller to handle org/repo:branch pools. Defaults to 20. Needs to be a
	// positive number.
	MaxGoroutines *int `json:"maxGoroutines,omitempty"`

	// Override the default method of merge. Valid options are squash, rebase, and merge.
	MergeType *string `json:"mergeMethod,omitempty"`

	// ContextOptions defines the default merge options. If not set it will infer
	// the required and optional contexts from the jobs configured and use the Git Provider
	// combined status; otherwise it may apply the branch protection setting or let user
	// define their own options in case branch protection is not used.
	ContextPolicy *ContextPolicy `json:"policy,omitempty"`
}

// RepoContextPolicy overrides the policy for repo, and any branch overrides.
type RepoContextPolicy struct {
	*ContextPolicy
	Branches *ReplaceableMapOfStringContextPolicy `json:"branches,omitempty"`
}

// ReplaceableMapOfStringContextPolicy is a map of context policies that can optionally completely replace any
// context policies defined in the parent scheduler
type ReplaceableMapOfStringContextPolicy struct {
	Replace bool `json:"replace,omitempty"`
	Items   map[string]*ContextPolicy
}

// ContextPolicy configures options about how to handle various contexts.
type ContextPolicy struct {
	// whether to consider unknown contexts optional (skip) or required.
	SkipUnknownContexts       *bool                      `json:"skipUnknownContexts,omitempty"`
	RequiredContexts          *ReplaceableSliceOfStrings `json:"required-contexts,omitempty"`
	RequiredIfPresentContexts *ReplaceableSliceOfStrings `json:"required-if-present-contexts,omitempty"`
	OptionalContexts          *ReplaceableSliceOfStrings `json:"optional-contexts,omitempty"`
	// Infer required and optional jobs from Branch Protection configuration
	FromBranchProtection *bool `json:"fromBranchProtection,omitempty"`
}

// Welcome welcome plugin config
type Welcome struct {
	MessageTemplate *string `json:"message_template,omitempty"`
}

// GlobalProtectionPolicy defines the default branch protection policy for the scheduler
type GlobalProtectionPolicy struct {
	// +optional
	*ProtectionPolicy
	ProtectTested *bool `json:"protectTested,omitempty"`
}

// ProtectionPolicy for merging.
type ProtectionPolicy struct {
	// Protect overrides whether branch protection is enabled if set.
	Protect *bool `json:"protect,omitempty"`
	// RequiredStatusChecks configures github contexts
	RequiredStatusChecks *BranchProtectionContextPolicy `json:"requiredStatusChecks,omitempty"`
	// Admins overrides whether protections apply to admins if set.
	Admins *bool `json:"enforceAdmins,omitempty"`
	// Restrictions limits who can merge
	Restrictions *Restrictions `json:"restrictions,omitempty"`
	// RequiredPullRequestReviews specifies approval/review criteria.
	RequiredPullRequestReviews *ReviewPolicy `json:"requiredPullRequestReviews,omitempty"`
}

// ReviewPolicy specifies git provider approval/review criteria.
// Any nil values inherit the policy from the parent, otherwise bool/ints are overridden.
// Non-empty lists are appended to parent lists.
type ReviewPolicy struct {
	// Restrictions appends users/teams that are allowed to merge
	DismissalRestrictions *Restrictions `json:"dismissalRestrictions,omitempty"`
	// DismissStale overrides whether new commits automatically dismiss old reviews if set
	DismissStale *bool `json:"dismissStaleReviews,omitempty"`
	// RequireOwners overrides whether CODEOWNERS must approve PRs if set
	RequireOwners *bool `json:"requireCodeOwnerReviews,omitempty"`
	// Approvals overrides the number of approvals required if set (set to 0 to disable)
	Approvals *int `json:"requiredApprovingReviewCount,omitempty"`
}

// Restrictions limits who can merge
// Users and Teams entries are appended to parent lists.
type Restrictions struct {
	Users *ReplaceableSliceOfStrings `json:"users"`
	Teams *ReplaceableSliceOfStrings `json:"teams"`
}

// BranchProtectionContextPolicy configures required git provider contexts.
// Strict determines whether merging to the branch invalidates existing contexts.
type BranchProtectionContextPolicy struct {
	// Contexts appends required contexts that must be green to merge
	Contexts *ReplaceableSliceOfStrings `json:"contexts,omitempty"`
	// Strict overrides whether new commits in the base branch require updating the PR if set
	Strict *bool `json:"strict,omitempty"`
}

// SchedulerAgent defines the scheduler agent configuration
type SchedulerAgent struct {
	// Agent defines the agent used to schedule jobs, by default Prow
	Agent *string `json:"agent"`
}

// ProtectionPolicies defines the branch protection policies
type ProtectionPolicies struct {
	// +optional
	*ProtectionPolicy
	// +optional
	Replace bool
	Items   map[string]*ProtectionPolicy `json:"entries,omitempty" protobuf:"bytes,1,opt,name=entries"`
}

// ReplaceableSliceOfExternalPlugins is a list of external plugins that can optionally completely replace the plugins
// in any parent SchedulerSpec
type ReplaceableSliceOfExternalPlugins struct {
	Replace bool
	Items   []*ExternalPlugin `json:"entries,omitempty" protobuf:"bytes,1,opt,name=entries"`
}

type Attachment struct {
	Name string   `json:"name,omitempty"  protobuf:"bytes,1,opt,name=name"`
	URLs []string `json:"urls,omitempty"  protobuf:"bytes,2,opt,name=urls"`
}
