package pipelinescheduler

import (
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	schedulerapi "github.com/jenkins-x-plugins/jx-gitops/pkg/apis/scheduler/v1alpha1"
	"github.com/jenkins-x/lighthouse-client/pkg/config"
	"github.com/jenkins-x/lighthouse-client/pkg/config/branchprotection"
	"github.com/jenkins-x/lighthouse-client/pkg/config/job"
	"github.com/jenkins-x/lighthouse-client/pkg/config/keeper"
	"github.com/jenkins-x/lighthouse-client/pkg/plugins"
	"github.com/pkg/errors"
	"github.com/rollout/rox-go/core/utils"
)

// BuildProwConfig takes a list of schedulers and creates a Prow Config from it
func BuildProwConfig(schedulers []*SchedulerLeaf) (*config.Config, *plugins.Configuration,
	error) {
	configResult := config.Config{
		JobConfig:  config.JobConfig{},
		ProwConfig: config.ProwConfig{},
	}
	pluginsResult := plugins.Configuration{}
	for _, scheduler := range schedulers {
		buildJobConfig(&configResult.JobConfig, &configResult.ProwConfig, scheduler.SchedulerSpec, scheduler.Org, scheduler.Repo)
		err := buildProwConfig(&configResult.ProwConfig, scheduler.SchedulerSpec, scheduler.Org, scheduler.Repo)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building ProwConfig for %v", scheduler)
		}
		buildPlugins(&pluginsResult, scheduler.SchedulerSpec, scheduler.Org, scheduler.Repo)
	}
	return &configResult, &pluginsResult, nil
}

func buildPlugins(answer *plugins.Configuration, scheduler *schedulerapi.SchedulerSpec, orgName,
	repoName string) {
	if scheduler.Plugins != nil {
		if answer.Plugins == nil {
			answer.Plugins = make(map[string][]string)
		}
		answer.Plugins[orgSlashRepo(orgName, repoName)] = scheduler.Plugins.Items
	}
	if answer.ExternalPlugins == nil {
		answer.ExternalPlugins = make(map[string][]plugins.ExternalPlugin)
	}

	if scheduler.ExternalPlugins != nil {
		var res []plugins.ExternalPlugin
		for _, src := range scheduler.ExternalPlugins.Items {
			if res == nil {
				res = make([]plugins.ExternalPlugin, 0)
			}
			externalPlugin := plugins.ExternalPlugin{}
			buildExternalPlugin(&externalPlugin, src)
			res = append(res, externalPlugin)
		}
		answer.ExternalPlugins[orgSlashRepo(orgName, repoName)] = res
	}
	if answer.Approve == nil {
		answer.Approve = make([]plugins.Approve, 0)
	}
	if scheduler.Approve != nil {
		approve := plugins.Approve{}
		buildApprove(&approve, scheduler.Approve, orgName, repoName)
		answer.Approve = append(answer.Approve, approve)
	}
	if scheduler.Welcome != nil {
		if answer.Welcome == nil {
			answer.Welcome = make([]plugins.Welcome, 0)
		}
		for _, welcome := range scheduler.Welcome {
			welcomeExists := false
			// TODO support Welcome.Repos
			for _, existingWelcome := range answer.Welcome {
				if *welcome.MessageTemplate == existingWelcome.MessageTemplate {
					welcomeExists = true
					break
				}
			}
			if !welcomeExists {
				answer.Welcome = append(answer.Welcome, plugins.Welcome{MessageTemplate: *welcome.MessageTemplate})
			}
		}
	}
	if scheduler.ConfigUpdater != nil {
		if answer.ConfigUpdater.Maps == nil {
			answer.ConfigUpdater.Maps = make(map[string]plugins.ConfigMapSpec)
			for key, value := range scheduler.ConfigUpdater.Map {
				answer.ConfigUpdater.Maps[key] = plugins.ConfigMapSpec{
					Name:                 value.Name,
					Namespace:            value.Namespace,
					Key:                  value.Key,
					AdditionalNamespaces: value.AdditionalNamespaces,
				}
			}

		}
		/* TODO removed
		if answer.ConfigUpdater.ConfigFile == "" {
			answer.ConfigUpdater.ConfigFile = scheduler.ConfigUpdater.ConfigFile
		}
		if answer.ConfigUpdater.PluginFile == "" {
			answer.ConfigUpdater.PluginFile = scheduler.ConfigUpdater.PluginFile
		}
		*/
	}
	if answer.Lgtm == nil {
		answer.Lgtm = make([]plugins.Lgtm, 0)
	}
	if scheduler.LGTM != nil {
		lgtm := plugins.Lgtm{}
		buildLgtm(&lgtm, scheduler.LGTM, orgName, repoName)
		answer.Lgtm = append(answer.Lgtm, lgtm)
	}
	if answer.Triggers == nil {
		answer.Triggers = make([]plugins.Trigger, 0)
	}
	if scheduler.Trigger != nil {
		trigger := plugins.Trigger{}
		buildTrigger(&trigger, scheduler.Trigger, orgName, repoName)
		answer.Triggers = append(answer.Triggers, trigger)
	}
}

func buildTrigger(answer *plugins.Trigger, trigger *schedulerapi.Trigger, orgName, repoName string) {
	if trigger.TrustedOrg != nil {
		answer.TrustedOrg = *trigger.TrustedOrg
	} else {
		answer.TrustedOrg = orgName
	}
	if trigger.TrustedApps != nil {
		answer.TrustedApps = trigger.TrustedApps
	}
	if trigger.OnlyOrgMembers != nil {
		answer.OnlyOrgMembers = *trigger.OnlyOrgMembers
	}
	if trigger.JoinOrgURL != nil {
		answer.JoinOrgURL = *trigger.JoinOrgURL
	}
	if trigger.IgnoreOkToTest != nil {
		answer.IgnoreOkToTest = *trigger.IgnoreOkToTest
	}
	if trigger.SkipDraftPR != nil {
		answer.SkipDraftPR = *trigger.SkipDraftPR
	}
	answer.Repos = []string{
		orgSlashRepo(orgName, repoName),
	}
}

func buildLgtm(answer *plugins.Lgtm, lgtm *schedulerapi.Lgtm, orgName, repoName string) {
	if lgtm.StickyLgtmTeam != nil {
		answer.StickyLgtmTeam = *lgtm.StickyLgtmTeam
	}
	if lgtm.ReviewActsAsLgtm != nil {
		answer.ReviewActsAsLgtm = *lgtm.ReviewActsAsLgtm
	}
	if lgtm.StoreTreeHash != nil {
		answer.StoreTreeHash = *lgtm.StoreTreeHash
	}
	answer.Repos = []string{
		orgSlashRepo(orgName, repoName),
	}
}

func buildApprove(answer *plugins.Approve, approve *schedulerapi.Approve, orgName, repoName string) {
	answer.IgnoreReviewState = approve.IgnoreReviewState
	answer.RequireSelfApproval = approve.RequireSelfApproval
	if approve.IssueRequired != nil {
		answer.IssueRequired = *approve.IssueRequired
	}
	if approve.LgtmActsAsApprove != nil {
		answer.LgtmActsAsApprove = *approve.LgtmActsAsApprove
	}
	answer.Repos = []string{
		orgSlashRepo(orgName, repoName),
	}
}

func buildExternalPlugin(answer *plugins.ExternalPlugin, plugin *schedulerapi.ExternalPlugin) {
	if plugin.Name != nil {
		answer.Name = *plugin.Name
	}
	if plugin.Endpoint != nil {
		answer.Endpoint = *plugin.Endpoint
	}
	if plugin.Events != nil {
		answer.Events = plugin.Events.Items
	}
}

func buildProwConfig(prowConfig *config.ProwConfig, scheduler *schedulerapi.SchedulerSpec, org, repo string) error {
	prowConfig.PushGateway.ServeMetrics = true
	if scheduler.Policy != nil {
		buildGlobalBranchProtection(&prowConfig.BranchProtection, scheduler.Policy)
	}
	if scheduler.Merger != nil {
		err := buildMerger(&prowConfig.Keeper, scheduler.Merger, org, repo)
		if err != nil {
			return errors.Wrapf(err, "building Merger for %v", scheduler)
		}
	}
	return nil
}

func buildPolicy(answer *branchprotection.Policy, policy *schedulerapi.ProtectionPolicy) {
	if policy.Protect != nil {
		answer.Protect = policy.Protect
	}
	if policy.Admins != nil {
		answer.Admins = policy.Admins
	}
	if policy.RequiredStatusChecks != nil {
		if answer.RequiredStatusChecks == nil {
			answer.RequiredStatusChecks = &branchprotection.ContextPolicy{}
		}
		buildBranchProtectionContextPolicy(answer.RequiredStatusChecks, policy.RequiredStatusChecks)
	}
	if policy.RequiredPullRequestReviews != nil {
		if answer.RequiredPullRequestReviews == nil {
			answer.RequiredPullRequestReviews = &branchprotection.ReviewPolicy{}
		}
		buildRequiredPullRequestReviews(answer.RequiredPullRequestReviews, policy.RequiredPullRequestReviews)
	}
	if policy.Restrictions != nil {
		if answer.Restrictions == nil {
			answer.Restrictions = &branchprotection.Restrictions{}
		}
		buildRestrictions(answer.Restrictions, policy.Restrictions)
	}
}

func buildBranchProtectionContextPolicy(answer *branchprotection.ContextPolicy,
	policy *schedulerapi.BranchProtectionContextPolicy) {
	if policy.Contexts != nil {
		if answer.Contexts == nil {
			answer.Contexts = make([]string, 0)
		}
		for _, i := range policy.Contexts.Items {
			found := false
			for _, existing := range answer.Contexts {
				if existing == i {
					found = true
					break
				}
			}
			if !found {
				answer.Contexts = append(answer.Contexts, i)
			}
		}
	}
	if policy.Strict != nil {
		answer.Strict = policy.Strict
	}
}

func buildRequiredPullRequestReviews(answer *branchprotection.ReviewPolicy, policy *schedulerapi.ReviewPolicy) {
	if policy.Approvals != nil {
		answer.Approvals = policy.Approvals
	}
	if policy.DismissStale != nil {
		answer.DismissStale = policy.DismissStale
	}
	if policy.RequireOwners != nil {
		answer.RequireOwners = policy.RequireOwners
	}
	if policy.DismissalRestrictions != nil {
		if answer.DismissalRestrictions == nil {
			answer.DismissalRestrictions = &branchprotection.Restrictions{}
		}
		buildRestrictions(answer.DismissalRestrictions, policy.DismissalRestrictions)
	}
}

func buildRestrictions(answer *branchprotection.Restrictions, restrictions *schedulerapi.Restrictions) {
	if restrictions.Users != nil {
		if answer.Users == nil {
			answer.Users = make([]string, 0)
		}
		answer.Users = append(answer.Users, restrictions.Users.Items...)
	}
	if restrictions.Teams != nil {
		if answer.Teams == nil {
			answer.Teams = make([]string, 0)
		}
		answer.Teams = append(answer.Teams, restrictions.Teams.Items...)
	}
}

func buildJobConfig(jobConfig *config.JobConfig, prowConfig *config.ProwConfig,
	scheduler *schedulerapi.SchedulerSpec, org, repo string) {
	if scheduler.Postsubmits != nil && scheduler.Postsubmits.Items != nil {
		buildPostsubmits(jobConfig, scheduler.Postsubmits.Items, org, repo)
	}
	if scheduler.Presubmits != nil && scheduler.Presubmits.Items != nil {
		buildPresubmits(jobConfig, scheduler.Presubmits.Items, org, repo)
	}
	if scheduler.Periodics != nil && len(scheduler.Periodics.Items) > 0 {
		buildPeriodics(jobConfig, scheduler.Periodics)
	}

	buildKeeperConfig(prowConfig, scheduler.Queries, scheduler.MergeMethod, scheduler.ProtectionPolicy, scheduler.ContextOptions, org, repo)

	if scheduler.Attachments != nil && len(scheduler.Attachments) > 0 {
		buildPlank(prowConfig, scheduler.Attachments)
	}
}

func buildPostsubmits(jobConfig *config.JobConfig, items []*job.Postsubmit, orgName, repoName string) {
	if jobConfig.Postsubmits == nil {
		jobConfig.Postsubmits = make(map[string][]job.Postsubmit)
	}
	orgSlashRepo := orgSlashRepo(orgName, repoName)
	for _, r := range items {
		if _, ok := jobConfig.Postsubmits[orgSlashRepo]; !ok {
			jobConfig.Postsubmits[orgSlashRepo] = make([]job.Postsubmit, 0)
		}
		jobConfig.Postsubmits[orgSlashRepo] = append(jobConfig.Postsubmits[orgSlashRepo], *r)
	}
}

func buildKeeperConfig(prowConfig *config.ProwConfig, queries []*schedulerapi.Query, mergeMethod *string, protectionPolicy *schedulerapi.ProtectionPolicies, contextOptions *schedulerapi.RepoContextPolicy, orgName, repoName string) {
	orgSlashRepo := orgSlashRepo(orgName, repoName)

	if len(queries) > 0 {
		buildQuery(&prowConfig.Keeper, queries, orgName, repoName)
	}

	if mergeMethod != nil {
		mt := keeper.PullRequestMergeType(*mergeMethod)
		if prowConfig.Keeper.MergeType == nil && mt != "" {
			prowConfig.Keeper.MergeType = make(map[string]keeper.PullRequestMergeType)
		}
		if mt != "" {
			prowConfig.Keeper.MergeType[orgSlashRepo] = mt
		}
	}
	if protectionPolicy != nil {
		if protectionPolicy.ProtectionPolicy != nil {
			buildBranchProtection(&prowConfig.BranchProtection, protectionPolicy.ProtectionPolicy,
				orgName, repoName, "")
		}
		for k, v := range protectionPolicy.Items {
			buildBranchProtection(&prowConfig.BranchProtection, v, orgName, repoName, k)
		}

	}
	if contextOptions != nil {
		policy := keeper.RepoContextPolicy{}
		buildRepoContextPolicy(&policy, contextOptions)
		if prowConfig.Keeper.ContextOptions.Orgs == nil {
			prowConfig.Keeper.ContextOptions.Orgs = make(map[string]keeper.OrgContextPolicy)
		}
		if _, ok := prowConfig.Keeper.ContextOptions.Orgs[orgName]; !ok {
			prowConfig.Keeper.ContextOptions.Orgs[orgName] = keeper.OrgContextPolicy{
				Repos: make(map[string]keeper.RepoContextPolicy),
			}
		}
		prowConfig.Keeper.ContextOptions.Orgs[orgName].Repos[repoName] = policy
	}
}

func buildPresubmits(jobConfig *config.JobConfig, items []*job.Presubmit, orgName, repoName string) {
	if jobConfig.Presubmits == nil {
		jobConfig.Presubmits = make(map[string][]job.Presubmit)
	}
	orgSlashRepo := orgSlashRepo(orgName, repoName)

	for _, r := range items {
		if _, ok := jobConfig.Presubmits[orgSlashRepo]; !ok {
			jobConfig.Presubmits[orgSlashRepo] = make([]job.Presubmit, 0)
		}
		jobConfig.Presubmits[orgSlashRepo] = append(jobConfig.Presubmits[orgSlashRepo], *r)
	}
}

func buildGlobalBranchProtection(answer *branchprotection.Config,
	globalProtectionPolicy *schedulerapi.GlobalProtectionPolicy) {
	if globalProtectionPolicy.ProtectTested != nil {
		answer.ProtectTested = *globalProtectionPolicy.ProtectTested
	}
	if globalProtectionPolicy.ProtectionPolicy != nil {
		buildBranchProtection(answer, globalProtectionPolicy.ProtectionPolicy, "", "", "")
	}
}

func buildBranchProtection(answer *branchprotection.Config,
	protectionPolicy *schedulerapi.ProtectionPolicy, orgName, repoName, branchName string) {
	if orgName != "" {
		if answer.Orgs == nil {
			answer.Orgs = make(map[string]branchprotection.Org)
		}
		if _, ok := answer.Orgs[orgName]; !ok {
			answer.Orgs[orgName] = branchprotection.Org{}
		}
		org := answer.Orgs[orgName]
		if repoName != "" {
			if org.Repos == nil {
				org.Repos = make(map[string]branchprotection.Repo)
			}
			if _, ok := answer.Orgs[orgName].Repos[repoName]; !ok {
				org.Repos[repoName] = branchprotection.Repo{}
			}
			repo := answer.Orgs[orgName].Repos[repoName]
			if branchName != "" {
				if repo.Branches == nil {
					repo.Branches = make(map[string]branchprotection.Branch)
				}
				if _, ok := repo.Branches[branchName]; !ok {
					repo.Branches[branchName] = branchprotection.Branch{}
				}
				branch := repo.Branches[branchName]
				buildPolicy(&branch.Policy, protectionPolicy)

			} else {
				buildPolicy(&repo.Policy, protectionPolicy)
			}
			org.Repos[repoName] = repo
		} else {
			buildPolicy(&org.Policy, protectionPolicy)
		}
		answer.Orgs[orgName] = org
	} else {
		buildPolicy(&answer.Policy, protectionPolicy)
	}
}

func orgSlashRepo(org, repo string) string {
	if repo == "" {
		return org
	}
	return fmt.Sprintf("%s/%s", org, repo)
}

func buildBase(answer, jobBase *job.Base) {
	if jobBase.Agent != "" {
		answer.Agent = jobBase.Agent
	}
	if jobBase.Labels != nil {
		answer.Labels = jobBase.Labels
	}
	if jobBase.MaxConcurrency <= 0 {
		answer.MaxConcurrency = jobBase.MaxConcurrency
	}
	if jobBase.Cluster != "" {
		answer.Cluster = jobBase.Cluster
	}
	if jobBase.Namespace != nil {
		answer.Namespace = jobBase.Namespace
	}
	if jobBase.Name != "" {
		answer.Name = jobBase.Name
	}
	if jobBase.Spec != nil {
		answer.Spec = jobBase.Spec
	}
}

func buildPlank(answer *config.ProwConfig, attachments []*schedulerapi.Attachment) {
	for attachmentIndex := range attachments {
		attachment := attachments[attachmentIndex]
		if attachment.Name == "reportTemplate" {
			answer.Plank.ReportTemplateString = attachment.URLs[0]
		}
	}
}

func buildPeriodics(answer *config.JobConfig, periodics *schedulerapi.Periodics) {
	if answer.Periodics == nil {
		answer.Periodics = make([]job.Periodic, 0)
	}
	for _, schedulerPeriodic := range periodics.Items {
		periodicAlreadyExists := false
		for existingPeriodicIndex := range answer.Periodics {
			if answer.Periodics[existingPeriodicIndex].Name == schedulerPeriodic.Name {
				periodicAlreadyExists = true
				break
			}
		}
		if !periodicAlreadyExists {
			periodic := job.Periodic{
				Cron: schedulerPeriodic.Cron,
			}
			buildBase(&periodic.Base, &schedulerPeriodic.Base)
			answer.Periodics = append(answer.Periodics, periodic)
		}
	}
}

func buildMerger(answer *keeper.Config, merger *schedulerapi.Merger, org, repo string) error {
	syncPeriod, err := merger.GetSyncPeriod()
	if err != nil {
		return errors.Wrapf(err, "failed to parse sync period")
	}
	if syncPeriod != nil {
		answer.SyncPeriod = *syncPeriod
	}
	if answer.SyncPeriod.Milliseconds() != 0 {
		answer.SyncPeriodString = answer.SyncPeriod.String()
	}

	if merger.StatusUpdatePeriod != nil {
		answer.StatusUpdatePeriod = *merger.StatusUpdatePeriod
	}
	if answer.StatusUpdatePeriod.Milliseconds() != 0 {
		answer.StatusUpdatePeriodString = answer.StatusUpdatePeriod.String()
	}

	if merger.TargetURL != nil {
		answer.TargetURL = *merger.TargetURL
	}
	if merger.PRStatusBaseURL != nil {
		answer.PRStatusBaseURL = *merger.PRStatusBaseURL
	}
	if merger.BlockerLabel != nil {
		answer.BlockerLabel = *merger.BlockerLabel
	}
	if merger.SquashLabel != nil {
		answer.SquashLabel = *merger.SquashLabel
	}
	if merger.MaxGoroutines != nil {
		answer.MaxGoroutines = *merger.MaxGoroutines
	}
	if merger.MergeType != nil {
		if answer.MergeType == nil {
			answer.MergeType = make(map[string]keeper.PullRequestMergeType)
		}
		answer.MergeType[fmt.Sprintf("%s/%s", org, repo)] = keeper.PullRequestMergeType(*merger.MergeType)
	}
	if merger.ContextPolicy != nil {
		buildContextPolicy(&answer.ContextOptions.ContextPolicy, merger.ContextPolicy)
	}
	return nil
}

func buildRepoContextPolicy(answer *keeper.RepoContextPolicy,
	repoContextPolicy *schedulerapi.RepoContextPolicy) {
	buildContextPolicy(&answer.ContextPolicy, repoContextPolicy.ContextPolicy)
	if repoContextPolicy.Branches != nil {
		for branch, policy := range repoContextPolicy.Branches.Items {
			if answer.Branches == nil {
				answer.Branches = make(map[string]keeper.ContextPolicy)
			}
			tidePolicy := keeper.ContextPolicy{}
			buildContextPolicy(&tidePolicy, policy)
			answer.Branches[branch] = tidePolicy
		}
	}
}

func buildContextPolicy(answer *keeper.ContextPolicy,
	contextOptions *schedulerapi.ContextPolicy) {
	if contextOptions != nil {
		if contextOptions.SkipUnknownContexts != nil {
			answer.SkipUnknownContexts = contextOptions.SkipUnknownContexts
		}
		if contextOptions.FromBranchProtection != nil {
			answer.FromBranchProtection = contextOptions.FromBranchProtection
		}
		if contextOptions.RequiredIfPresentContexts != nil {
			answer.RequiredIfPresentContexts = contextOptions.RequiredIfPresentContexts.Items
		}
		if contextOptions.RequiredContexts != nil {
			answer.RequiredContexts = contextOptions.RequiredContexts.Items
		}
		if contextOptions.OptionalContexts != nil {
			answer.OptionalContexts = contextOptions.OptionalContexts.Items
		}
	}
}

func buildQuery(answer *keeper.Config, queries []*schedulerapi.Query, org, repo string) {
	if answer.Queries == nil {
		answer.Queries = keeper.Queries{}
	}
	tideQuery := &keeper.Query{
		Repos: []string{orgSlashRepo(org, repo)},
	}
	for _, query := range queries {
		if query.ExcludedBranches != nil {
			tideQuery.ExcludedBranches = query.ExcludedBranches.Items
		}
		if query.IncludedBranches != nil {
			tideQuery.IncludedBranches = query.IncludedBranches.Items
		}
		if query.Labels != nil {
			tideQuery.Labels = query.Labels.Items
		}
		if query.MissingLabels != nil {
			tideQuery.MissingLabels = query.MissingLabels.Items
		}
		if query.Milestone != nil {
			tideQuery.Milestone = *query.Milestone
		}
		if query.ReviewApprovedRequired != nil {
			tideQuery.ReviewApprovedRequired = *query.ReviewApprovedRequired
		}
		mergedWithExisting := false
		for index := range answer.Queries {
			existingQuery := &answer.Queries[index]
			if cmp.Equal(existingQuery, tideQuery, cmpopts.IgnoreFields(keeper.Query{}, "Repos")) {
				mergedWithExisting = true
				for _, newRepo := range tideQuery.Repos {
					if !utils.ContainsString(existingQuery.Repos, newRepo) {
						existingQuery.Repos = append(existingQuery.Repos, newRepo)
					}
				}
			}
		}
		if !mergedWithExisting {
			answer.Queries = append(answer.Queries, *tideQuery)
		}
	}
}
