---
title: API Documentation
linktitle: API Documentation
description: Reference of the jx-promote configuration
weight: 10
---
<p>Packages:</p>
<ul>
<li>
<a href="#scheduler.jenkins-x.io%2fv1alpha1">scheduler.jenkins-x.io/v1alpha1</a>
</li>
</ul>
<h2 id="scheduler.jenkins-x.io/v1alpha1">scheduler.jenkins-x.io/v1alpha1</h2>
<p>
<p>Package v1alpha1 is the v1alpha1 version of the API.</p>
</p>
Resource Types:
<ul><li>
<a href="#scheduler.jenkins-x.io/v1alpha1.Scheduler">Scheduler</a>
</li></ul>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Scheduler">Scheduler
</h3>
<p>
<p>Scheduler is configuration for a pipeline scheduler</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
scheduler.jenkins-x.io/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>Scheduler</code></td>
</tr>
<tr>
<td>
<code>metadata</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.13/#objectmeta-v1-meta">
Kubernetes meta/v1.ObjectMeta
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Standard object&rsquo;s metadata.
More info: <a href="https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata">https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata</a></p>
Refer to the Kubernetes API documentation for the fields of the
<code>metadata</code> field.
</td>
</tr>
<tr>
<td>
<code>spec</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">
SchedulerSpec
</a>
</em>
</td>
<td>
<br/>
<br/>
<table>
<tr>
<td>
<code>schedulerAgent</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerAgent">
SchedulerAgent
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>policy</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.GlobalProtectionPolicy">
GlobalProtectionPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>presubmits</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Presubmits">
Presubmits
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>postsubmits</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Postsubmits">
Postsubmits
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>queries</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.*./pkg/apis/scheduler/v1alpha1.Query">
[]*./pkg/apis/scheduler/v1alpha1.Query
</a>
</em>
</td>
<td>
<p>Queries add keeper queries</p>
</td>
</tr>
<tr>
<td>
<code>merge_method</code></br>
<em>
string
</em>
</td>
<td>
<p>MergeMethod Override the default method of merge. Valid options are squash, rebase, and merge.</p>
</td>
</tr>
<tr>
<td>
<code>protection_policy</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ProtectionPolicies">
ProtectionPolicies
</a>
</em>
</td>
<td>
<p>ProtectionPolicy the protection policy</p>
</td>
</tr>
<tr>
<td>
<code>context_options</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.RepoContextPolicy">
RepoContextPolicy
</a>
</em>
</td>
<td>
<p>ContextOptions defines the merge options. If not set it will infer
the required and optional contexts from the jobs configured and use the Git Provider
combined status; otherwise it may apply the branch protection setting or let user
define their own options in case branch protection is not used.</p>
</td>
</tr>
<tr>
<td>
<code>trigger</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Trigger">
Trigger
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>approve</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Approve">
Approve
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lgtm</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Lgtm">
Lgtm
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>external_plugins</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfExternalPlugins">
ReplaceableSliceOfExternalPlugins
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>merger</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Merger">
Merger
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>plugins</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
<p>Plugins is a list of plugin names enabled for a repo</p>
</td>
</tr>
<tr>
<td>
<code>config_updater</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ConfigUpdater">
ConfigUpdater
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>welcome</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.*./pkg/apis/scheduler/v1alpha1.Welcome">
[]*./pkg/apis/scheduler/v1alpha1.Welcome
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>periodics</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Periodics">
Periodics
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>attachments</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.*./pkg/apis/scheduler/v1alpha1.Attachment">
[]*./pkg/apis/scheduler/v1alpha1.Attachment
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>in_repo</code></br>
<em>
bool
</em>
</td>
<td>
<p>InRepo if enabled specifies that the repositories using this scheduler will enable in-repo</p>
</td>
</tr>
</table>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Approve">Approve
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>Approve specifies a configuration for a single approve.</p>
<p>The configuration for the approve plugin is defined as a list of these structures.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>issue_required</code></br>
<em>
bool
</em>
</td>
<td>
<p>IssueRequired indicates if an associated issue is required for approval in
the specified repos.</p>
</td>
</tr>
<tr>
<td>
<code>require_self_approval</code></br>
<em>
bool
</em>
</td>
<td>
<p>RequireSelfApproval requires PR authors to explicitly approve their PRs.
Otherwise the plugin assumes the author of the PR approves the changes in the PR.</p>
</td>
</tr>
<tr>
<td>
<code>lgtm_acts_as_approve</code></br>
<em>
bool
</em>
</td>
<td>
<p>LgtmActsAsApprove indicates that the lgtm command should be used to
indicate approval</p>
</td>
</tr>
<tr>
<td>
<code>ignore_review_state</code></br>
<em>
bool
</em>
</td>
<td>
<p>IgnoreReviewState causes the approve plugin to ignore the GitHub review state. Otherwise:
* an APPROVE github review is equivalent to leaving an &ldquo;/approve&rdquo; message.
* A REQUEST_CHANGES github review is equivalent to leaving an /approve cancel&rdquo; message.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Attachment">Attachment
</h3>
<p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>urls</code></br>
<em>
[]string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.BranchProtectionContextPolicy">BranchProtectionContextPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ProtectionPolicy">ProtectionPolicy</a>)
</p>
<p>
<p>BranchProtectionContextPolicy configures required git provider contexts.
Strict determines whether merging to the branch invalidates existing contexts.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>contexts</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
<p>Contexts appends required contexts that must be green to merge</p>
</td>
</tr>
<tr>
<td>
<code>strict</code></br>
<em>
bool
</em>
</td>
<td>
<p>Strict overrides whether new commits in the base branch require updating the PR if set</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ConfigMapSpec">ConfigMapSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ConfigUpdater">ConfigUpdater</a>)
</p>
<p>
<p>ConfigMapSpec contains configuration options for the configMap being updated
by the config-updater plugin.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of ConfigMap</p>
</td>
</tr>
<tr>
<td>
<code>key</code></br>
<em>
string
</em>
</td>
<td>
<p>Key is the key in the ConfigMap to update with the file contents.
If no explicit key is given, the basename of the file will be used.</p>
</td>
</tr>
<tr>
<td>
<code>namespace</code></br>
<em>
string
</em>
</td>
<td>
<p>Namespace in which the configMap needs to be deployed. If no namespace is specified
it will be deployed to the ProwJobNamespace.</p>
</td>
</tr>
<tr>
<td>
<code>additional_namespaces</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Namespaces in which the configMap needs to be deployed, in addition to the above
namespace provided, or the default if it is not set.</p>
</td>
</tr>
<tr>
<td>
<code>-</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Namespaces is the fully resolved list of Namespaces to deploy the ConfigMap in</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ConfigUpdater">ConfigUpdater
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>ConfigUpdater holds configuration for the config updater plugin</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>map</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ConfigMapSpec">
map[string]./pkg/apis/scheduler/v1alpha1.ConfigMapSpec
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>config_file</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>plugin_file</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ConfigMap</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ConfigMapSpec">
ConfigMapSpec
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ContextPolicy">ContextPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Merger">Merger</a>, 
<a href="#scheduler.jenkins-x.io/v1alpha1.RepoContextPolicy">RepoContextPolicy</a>)
</p>
<p>
<p>ContextPolicy configures options about how to handle various contexts.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>skip-unknown-contexts</code></br>
<em>
bool
</em>
</td>
<td>
<p>whether to consider unknown contexts optional (skip) or required.</p>
</td>
</tr>
<tr>
<td>
<code>required-contexts</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>required-if-present-contexts</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>optional-contexts</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>from-branch-protection</code></br>
<em>
bool
</em>
</td>
<td>
<p>Infer required and optional jobs from Branch Protection configuration</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ExternalPlugin">ExternalPlugin
</h3>
<p>
<p>ExternalPlugin holds configuration for registering an external
plugin.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name of the plugin.</p>
</td>
</tr>
<tr>
<td>
<code>endpoint</code></br>
<em>
string
</em>
</td>
<td>
<p>Endpoint is the location of the external plugin. Defaults to
the name of the plugin, ie. &ldquo;http://{{name}}&rdquo;.</p>
</td>
</tr>
<tr>
<td>
<code>events</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
<p>ReplaceableSliceOfStrings are the events that need to be demuxed by the hook
server to the external plugin. If no events are specified,
everything is sent.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.GlobalProtectionPolicy">GlobalProtectionPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>GlobalProtectionPolicy defines the default branch protection policy for the scheduler</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ProtectionPolicy</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ProtectionPolicy">
ProtectionPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>protect_tested</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Lgtm">Lgtm
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>Lgtm specifies a configuration for a single lgtm.
The configuration for the lgtm plugin is defined as a list of these structures.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>review_acts_as_lgtm</code></br>
<em>
bool
</em>
</td>
<td>
<p>ReviewActsAsLgtm indicates that a Github review of &ldquo;approve&rdquo; or &ldquo;request changes&rdquo;
acts as adding or removing the lgtm label</p>
</td>
</tr>
<tr>
<td>
<code>store_tree_hash</code></br>
<em>
bool
</em>
</td>
<td>
<p>StoreTreeHash indicates if tree_hash should be stored inside a comment to detect
squashed commits before removing lgtm labels</p>
</td>
</tr>
<tr>
<td>
<code>trusted_team_for_sticky_lgtm</code></br>
<em>
string
</em>
</td>
<td>
<p>WARNING: This disables the security mechanism that prevents a malicious member (or
compromised GitHub account) from merging arbitrary code. Use with caution.</p>
<p>StickyLgtmTeam specifies the Github team whose members are trusted with sticky LGTM,
which eliminates the need to re-lgtm minor fixes/updates.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Merger">Merger
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>Merger defines the options used to merge the PR</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>sync_period</code></br>
<em>
string
</em>
</td>
<td>
<p>SyncPeriodString compiles into SyncPeriod at load time.</p>
</td>
</tr>
<tr>
<td>
<code>-</code></br>
<em>
time.Duration
</em>
</td>
<td>
<p>SyncPeriod specifies how often Merger will sync jobs with Github. Defaults to 1m.</p>
</td>
</tr>
<tr>
<td>
<code>status_update_period</code></br>
<em>
string
</em>
</td>
<td>
<p>StatusUpdatePeriodString compiles into StatusUpdatePeriod at load time.</p>
</td>
</tr>
<tr>
<td>
<code>-</code></br>
<em>
time.Duration
</em>
</td>
<td>
<p>StatusUpdatePeriod</p>
</td>
</tr>
<tr>
<td>
<code>target_url</code></br>
<em>
string
</em>
</td>
<td>
<p>URL for status contexts.</p>
</td>
</tr>
<tr>
<td>
<code>pr_status_base_url</code></br>
<em>
string
</em>
</td>
<td>
<p>PRStatusBaseURL is the base URL for the PR status page.
This is used to link to a merge requirements overview
in the status context.</p>
</td>
</tr>
<tr>
<td>
<code>blocker_label</code></br>
<em>
string
</em>
</td>
<td>
<p>BlockerLabel is an optional label that is used to identify merge blocking
Git Provider issues.
Leave this blank to disable this feature and save 1 API token per sync loop.</p>
</td>
</tr>
<tr>
<td>
<code>squash_label</code></br>
<em>
string
</em>
</td>
<td>
<p>SquashLabel is an optional label that is used to identify PRs that should
always be squash merged.
Leave this blank to disable this feature.</p>
</td>
</tr>
<tr>
<td>
<code>max_goroutines</code></br>
<em>
int
</em>
</td>
<td>
<p>MaxGoroutines is the maximum number of goroutines spawned inside the
controller to handle org/repo:branch pools. Defaults to 20. Needs to be a
positive number.</p>
</td>
</tr>
<tr>
<td>
<code>merge_method</code></br>
<em>
string
</em>
</td>
<td>
<p>Override the default method of merge. Valid options are squash, rebase, and merge.</p>
</td>
</tr>
<tr>
<td>
<code>policy</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ContextPolicy">
ContextPolicy
</a>
</em>
</td>
<td>
<p>ContextOptions defines the default merge options. If not set it will infer
the required and optional contexts from the jobs configured and use the Git Provider
combined status; otherwise it may apply the branch protection setting or let user
define their own options in case branch protection is not used.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Periodics">Periodics
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>Periodics is a list of jobs to be run periodically</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>entries</code></br>
<em>
[]*github.com/jenkins-x/lighthouse-client/pkg/config/job.Periodic
</em>
</td>
<td>
<p>Items are the post submit configurations</p>
</td>
</tr>
<tr>
<td>
<code>replace</code></br>
<em>
bool
</em>
</td>
<td>
<p>Replace the existing entries</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Postsubmits">Postsubmits
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>Postsubmits is a list of Postsubmit job configurations that can optionally completely replace the Postsubmit job
configurations in the parent scheduler</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>entries</code></br>
<em>
[]*github.com/jenkins-x/lighthouse-client/pkg/config/job.Postsubmit
</em>
</td>
<td>
<p>Items are the post submit configurations</p>
</td>
</tr>
<tr>
<td>
<code>replace</code></br>
<em>
bool
</em>
</td>
<td>
<p>Replace the existing entries</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Presubmits">Presubmits
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>Presubmits is a list of Presubmit job configurations that can optionally completely replace the Presubmit job
configurations in the parent scheduler</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>entries</code></br>
<em>
[]*github.com/jenkins-x/lighthouse-client/pkg/config/job.Presubmit
</em>
</td>
<td>
<p>Items are the Presubmit configurtations</p>
</td>
</tr>
<tr>
<td>
<code>replace</code></br>
<em>
bool
</em>
</td>
<td>
<p>Replace the existing entries</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ProtectionPolicies">ProtectionPolicies
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>ProtectionPolicies defines the branch protection policies</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ProtectionPolicy</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ProtectionPolicy">
ProtectionPolicy
</a>
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>Replace</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
</td>
</tr>
<tr>
<td>
<code>entries</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.*./pkg/apis/scheduler/v1alpha1.ProtectionPolicy">
map[string]*./pkg/apis/scheduler/v1alpha1.ProtectionPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ProtectionPolicy">ProtectionPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.GlobalProtectionPolicy">GlobalProtectionPolicy</a>, 
<a href="#scheduler.jenkins-x.io/v1alpha1.ProtectionPolicies">ProtectionPolicies</a>)
</p>
<p>
<p>ProtectionPolicy for merging.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>protect</code></br>
<em>
bool
</em>
</td>
<td>
<p>Protect overrides whether branch protection is enabled if set.</p>
</td>
</tr>
<tr>
<td>
<code>required_status_checks</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.BranchProtectionContextPolicy">
BranchProtectionContextPolicy
</a>
</em>
</td>
<td>
<p>RequiredStatusChecks configures github contexts</p>
</td>
</tr>
<tr>
<td>
<code>enforce_admins</code></br>
<em>
bool
</em>
</td>
<td>
<p>Admins overrides whether protections apply to admins if set.</p>
</td>
</tr>
<tr>
<td>
<code>restrictions</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Restrictions">
Restrictions
</a>
</em>
</td>
<td>
<p>Restrictions limits who can merge</p>
</td>
</tr>
<tr>
<td>
<code>required_pull_request_reviews</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReviewPolicy">
ReviewPolicy
</a>
</em>
</td>
<td>
<p>RequiredPullRequestReviews specifies approval/review criteria.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.PullRequestMergeType">PullRequestMergeType
(<code>string</code> alias)</p></h3>
<p>
<p>PullRequestMergeType enumerates the types of merges the Git Provider API can
perform
<a href="https://developer.github.com/v3/pulls/#merge-a-pull-request-merge-button">https://developer.github.com/v3/pulls/#merge-a-pull-request-merge-button</a></p>
</p>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Query">Query
</h3>
<p>
<p>Query is turned into a Git Provider search query. See the docs for details:
<a href="https://help.github.com/articles/searching-issues-and-pull-requests/">https://help.github.com/articles/searching-issues-and-pull-requests/</a></p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>excludedBranches</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>included_branches</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>labels</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>missingLabels</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>milestone</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>review_approved_required</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ReplaceableMapOfStringContextPolicy">ReplaceableMapOfStringContextPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.RepoContextPolicy">RepoContextPolicy</a>)
</p>
<p>
<p>ReplaceableMapOfStringContextPolicy is a map of context policies that can optionally completely replace any
context policies defined in the parent scheduler</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>replace</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>Items</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.*./pkg/apis/scheduler/v1alpha1.ContextPolicy">
map[string]*./pkg/apis/scheduler/v1alpha1.ContextPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfExternalPlugins">ReplaceableSliceOfExternalPlugins
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>ReplaceableSliceOfExternalPlugins is a list of external plugins that can optionally completely replace the plugins
in any parent SchedulerSpec</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>Replace</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>entries</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.*./pkg/apis/scheduler/v1alpha1.ExternalPlugin">
[]*./pkg/apis/scheduler/v1alpha1.ExternalPlugin
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">ReplaceableSliceOfStrings
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.BranchProtectionContextPolicy">BranchProtectionContextPolicy</a>, 
<a href="#scheduler.jenkins-x.io/v1alpha1.ContextPolicy">ContextPolicy</a>, 
<a href="#scheduler.jenkins-x.io/v1alpha1.ExternalPlugin">ExternalPlugin</a>, 
<a href="#scheduler.jenkins-x.io/v1alpha1.Query">Query</a>, 
<a href="#scheduler.jenkins-x.io/v1alpha1.Restrictions">Restrictions</a>, 
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>ReplaceableSliceOfStrings is a slice of strings that can optionally completely replace the slice of strings
defined in the parent scheduler</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>entries</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Items is the string values</p>
</td>
</tr>
<tr>
<td>
<code>replace</code></br>
<em>
bool
</em>
</td>
<td>
<p>Replace the existing entries</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.RepoContextPolicy">RepoContextPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>RepoContextPolicy overrides the policy for repo, and any branch overrides.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>ContextPolicy</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ContextPolicy">
ContextPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>branches</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableMapOfStringContextPolicy">
ReplaceableMapOfStringContextPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Restrictions">Restrictions
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ProtectionPolicy">ProtectionPolicy</a>, 
<a href="#scheduler.jenkins-x.io/v1alpha1.ReviewPolicy">ReviewPolicy</a>)
</p>
<p>
<p>Restrictions limits who can merge
Users and Teams entries are appended to parent lists.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>users</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>teams</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.ReviewPolicy">ReviewPolicy
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ProtectionPolicy">ProtectionPolicy</a>)
</p>
<p>
<p>ReviewPolicy specifies git provider approval/review criteria.
Any nil values inherit the policy from the parent, otherwise bool/ints are overridden.
Non-empty lists are appended to parent lists.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>dismissal_restrictions</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Restrictions">
Restrictions
</a>
</em>
</td>
<td>
<p>Restrictions appends users/teams that are allowed to merge</p>
</td>
</tr>
<tr>
<td>
<code>dismiss_stale_reviews</code></br>
<em>
bool
</em>
</td>
<td>
<p>DismissStale overrides whether new commits automatically dismiss old reviews if set</p>
</td>
</tr>
<tr>
<td>
<code>require_code_owner_reviews</code></br>
<em>
bool
</em>
</td>
<td>
<p>RequireOwners overrides whether CODEOWNERS must approve PRs if set</p>
</td>
</tr>
<tr>
<td>
<code>required_approving_review_count</code></br>
<em>
int
</em>
</td>
<td>
<p>Approvals overrides the number of approvals required if set (set to 0 to disable)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.SchedulerAgent">SchedulerAgent
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>SchedulerAgent defines the scheduler agent configuration</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>agent</code></br>
<em>
string
</em>
</td>
<td>
<p>Agent defines the agent used to schedule jobs, by default Prow</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Scheduler">Scheduler</a>)
</p>
<p>
<p>SchedulerSpec defines the pipeline scheduler (e.g. Prow) configuration</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>schedulerAgent</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerAgent">
SchedulerAgent
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>policy</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.GlobalProtectionPolicy">
GlobalProtectionPolicy
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>presubmits</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Presubmits">
Presubmits
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>postsubmits</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Postsubmits">
Postsubmits
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>queries</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.*./pkg/apis/scheduler/v1alpha1.Query">
[]*./pkg/apis/scheduler/v1alpha1.Query
</a>
</em>
</td>
<td>
<p>Queries add keeper queries</p>
</td>
</tr>
<tr>
<td>
<code>merge_method</code></br>
<em>
string
</em>
</td>
<td>
<p>MergeMethod Override the default method of merge. Valid options are squash, rebase, and merge.</p>
</td>
</tr>
<tr>
<td>
<code>protection_policy</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ProtectionPolicies">
ProtectionPolicies
</a>
</em>
</td>
<td>
<p>ProtectionPolicy the protection policy</p>
</td>
</tr>
<tr>
<td>
<code>context_options</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.RepoContextPolicy">
RepoContextPolicy
</a>
</em>
</td>
<td>
<p>ContextOptions defines the merge options. If not set it will infer
the required and optional contexts from the jobs configured and use the Git Provider
combined status; otherwise it may apply the branch protection setting or let user
define their own options in case branch protection is not used.</p>
</td>
</tr>
<tr>
<td>
<code>trigger</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Trigger">
Trigger
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>approve</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Approve">
Approve
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>lgtm</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Lgtm">
Lgtm
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>external_plugins</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfExternalPlugins">
ReplaceableSliceOfExternalPlugins
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>merger</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Merger">
Merger
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>plugins</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ReplaceableSliceOfStrings">
ReplaceableSliceOfStrings
</a>
</em>
</td>
<td>
<p>Plugins is a list of plugin names enabled for a repo</p>
</td>
</tr>
<tr>
<td>
<code>config_updater</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.ConfigUpdater">
ConfigUpdater
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>welcome</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.*./pkg/apis/scheduler/v1alpha1.Welcome">
[]*./pkg/apis/scheduler/v1alpha1.Welcome
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>periodics</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.Periodics">
Periodics
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>attachments</code></br>
<em>
<a href="#scheduler.jenkins-x.io/v1alpha1.*./pkg/apis/scheduler/v1alpha1.Attachment">
[]*./pkg/apis/scheduler/v1alpha1.Attachment
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>in_repo</code></br>
<em>
bool
</em>
</td>
<td>
<p>InRepo if enabled specifies that the repositories using this scheduler will enable in-repo</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Trigger">Trigger
</h3>
<p>
(<em>Appears on:</em>
<a href="#scheduler.jenkins-x.io/v1alpha1.SchedulerSpec">SchedulerSpec</a>)
</p>
<p>
<p>Trigger specifies a configuration for a single trigger.</p>
<p>The configuration for the trigger plugin is defined as a list of these structures.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>trusted_org</code></br>
<em>
string
</em>
</td>
<td>
<p>TrustedOrg is the org whose members&rsquo; PRs will be automatically built
for PRs to the above repos. The default is the PR&rsquo;s org.</p>
</td>
</tr>
<tr>
<td>
<code>join_org_url</code></br>
<em>
string
</em>
</td>
<td>
<p>JoinOrgURL is a link that redirects users to a location where they
should be able to read more about joining the organization in order
to become trusted members. Defaults to the Github link of TrustedOrg.</p>
</td>
</tr>
<tr>
<td>
<code>only_org_members</code></br>
<em>
bool
</em>
</td>
<td>
<p>OnlyOrgMembers requires PRs and/or /ok-to-test comments to come from org members.
By default, trigger also include repo collaborators.</p>
</td>
</tr>
<tr>
<td>
<code>ignore_ok_to_test</code></br>
<em>
bool
</em>
</td>
<td>
<p>IgnoreOkToTest makes trigger ignore /ok-to-test comments.
This is a security mitigation to only allow testing from trusted users.</p>
</td>
</tr>
<tr>
<td>
<code>elide_skipped_contexts</code></br>
<em>
bool
</em>
</td>
<td>
<p>ElideSkippedContexts makes trigger not post &ldquo;Skipped&rdquo; contexts for jobs
that could run but do not run.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="scheduler.jenkins-x.io/v1alpha1.Welcome">Welcome
</h3>
<p>
<p>Welcome welcome plugin config</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>message_template</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <code>gen-crd-api-reference-docs</code>
on git commit <code>f5285777</code>.
</em></p>
