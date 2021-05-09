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
		err := buildJobConfig(&configResult.JobConfig, &configResult.ProwConfig, scheduler.SchedulerSpec, scheduler.Org, scheduler.Repo)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building JobConfig for %v", scheduler)
		}
		err = buildProwConfig(&configResult.ProwConfig, scheduler.SchedulerSpec, scheduler.Org, scheduler.Repo)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building ProwConfig for %v", scheduler)
		}
		err = buildPlugins(&pluginsResult, scheduler.SchedulerSpec, scheduler.Org, scheduler.Repo)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "building Plugins for %v", scheduler)
		}
	}
	return &configResult, &pluginsResult, nil
}

func buildPlugins(answer *plugins.Configuration, scheduler *schedulerapi.SchedulerSpec, orgName string,
	repoName string) error {
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
			err := buildExternalPlugin(&externalPlugin, src)
			if err != nil {
				return errors.Wrapf(err, "building ExternalPlugin for %v", src)
			}
			res = append(res, externalPlugin)
		}
		answer.ExternalPlugins[orgSlashRepo(orgName, repoName)] = res
	}
	if answer.Approve == nil {
		answer.Approve = make([]plugins.Approve, 0)
	}
	if scheduler.Approve != nil {
		approve := plugins.Approve{}
		err := buildApprove(&approve, scheduler.Approve, orgName, repoName)
		if err != nil {
			return errors.Wrapf(err, "building Approve for %v", scheduler.Approve)
		}
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
		err := buildLgtm(&lgtm, scheduler.LGTM, orgName, repoName)
		if err != nil {
			return errors.Wrapf(err, "building LGTM for %v", scheduler.LGTM)
		}
		answer.Lgtm = append(answer.Lgtm, lgtm)
	}
	if answer.Triggers == nil {
		answer.Triggers = make([]plugins.Trigger, 0)
	}
	if scheduler.Trigger != nil {
		trigger := plugins.Trigger{}
		err := buildTrigger(&trigger, scheduler.Trigger, orgName, repoName)
		if err != nil {
			return errors.Wrapf(err, "building Triggers for %v", scheduler.Trigger)
		}
		answer.Triggers = append(answer.Triggers, trigger)
	}
	return nil
}

func buildTrigger(answer *plugins.Trigger, trigger *schedulerapi.Trigger, orgName string, repoName string) error {
	if trigger.TrustedOrg != nil {
		answer.TrustedOrg = *trigger.TrustedOrg
	} else {
		answer.TrustedOrg = orgName
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
	answer.Repos = []string{
		orgSlashRepo(orgName, repoName),
	}
	return nil
}

func buildLgtm(answer *plugins.Lgtm, lgtm *schedulerapi.Lgtm, orgName string, repoName string) error {
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
	return nil
}

func buildApprove(answer *plugins.Approve, approve *schedulerapi.Approve, orgName string, repoName string) error {
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
	return nil
}

func buildExternalPlugin(answer *plugins.ExternalPlugin, plugin *schedulerapi.ExternalPlugin) error {
	if plugin.Name != nil {
		answer.Name = *plugin.Name
	}
	if plugin.Endpoint != nil {
		answer.Endpoint = *plugin.Endpoint
	}
	if plugin.Events != nil {
		answer.Events = plugin.Events.Items
	}
	return nil
}

func buildProwConfig(prowConfig *config.ProwConfig, scheduler *schedulerapi.SchedulerSpec, org string, repo string) error {
	prowConfig.PushGateway.ServeMetrics = true
	if scheduler.Policy != nil {
		err := buildGlobalBranchProtection(&prowConfig.BranchProtection, scheduler.Policy)
		if err != nil {
			return errors.Wrapf(err, "building BranchProtection for %v", scheduler)
		}
	}
	if scheduler.Merger != nil {
		err := buildMerger(&prowConfig.Keeper, scheduler.Merger, org, repo)
		if err != nil {
			return errors.Wrapf(err, "building Merger for %v", scheduler)
		}
	}
	return nil
}

func buildPolicy(answer *branchprotection.Policy, policy *schedulerapi.ProtectionPolicy) error {
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
		err := buildBranchProtectionContextPolicy(answer.RequiredStatusChecks, policy.RequiredStatusChecks)
		if err != nil {
			return errors.Wrapf(err, "building ContextPolicy from %v", policy)
		}
	}
	if policy.RequiredPullRequestReviews != nil {
		if answer.RequiredPullRequestReviews == nil {
			answer.RequiredPullRequestReviews = &branchprotection.ReviewPolicy{}
		}
		err := buildRequiredPullRequestReviews(answer.RequiredPullRequestReviews, policy.RequiredPullRequestReviews)
		if err != nil {
			return errors.Wrapf(err, "building RequiredPullRequestReviews from %v", policy)
		}
	}
	if policy.Restrictions != nil {
		if answer.Restrictions == nil {
			answer.Restrictions = &branchprotection.Restrictions{}
		}
		err := buildRestrictions(answer.Restrictions, policy.Restrictions)
		if err != nil {
			return errors.Wrapf(err, "building Restrictions from %v", policy)
		}
	}
	return nil
}

func buildBranchProtectionContextPolicy(answer *branchprotection.ContextPolicy,
	policy *schedulerapi.BranchProtectionContextPolicy) error {
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
	return nil
}

func buildRequiredPullRequestReviews(answer *branchprotection.ReviewPolicy, policy *schedulerapi.ReviewPolicy) error {
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
		err := buildRestrictions(answer.DismissalRestrictions, policy.DismissalRestrictions)
		if err != nil {
			return errors.Wrapf(err, "building DismissalRestrictions from %v", policy)
		}
	}
	return nil
}

func buildRestrictions(answer *branchprotection.Restrictions, restrictions *schedulerapi.Restrictions) error {
	if restrictions.Users != nil {
		if answer.Users == nil {
			answer.Users = make([]string, 0)
		}
		for _, i := range restrictions.Users.Items {
			answer.Users = append(answer.Users, i)
		}
	}
	if restrictions.Teams != nil {
		if answer.Teams == nil {
			answer.Teams = make([]string, 0)
		}
		for _, i := range restrictions.Teams.Items {
			answer.Teams = append(answer.Teams, i)
		}
	}
	return nil
}

func buildJobConfig(jobConfig *config.JobConfig, prowConfig *config.ProwConfig,
	scheduler *schedulerapi.SchedulerSpec, org string, repo string) error {
	if scheduler.Postsubmits != nil && scheduler.Postsubmits.Items != nil {
		err := buildPostsubmits(jobConfig, scheduler.Postsubmits.Items, org, repo)
		if err != nil {
			return errors.Wrapf(err, "building Postsubmits from %v", scheduler)
		}
	}
	if scheduler.Presubmits != nil && scheduler.Presubmits.Items != nil {
		err := buildPresubmits(jobConfig, scheduler.Presubmits.Items, org, repo)
		if err != nil {
			return errors.Wrapf(err, "building Presubmits from %v", scheduler)
		}
	}
	if scheduler.Periodics != nil && len(scheduler.Periodics.Items) > 0 {
		err := buildPeriodics(jobConfig, scheduler.Periodics)
		if err != nil {
			return errors.Wrapf(err, "building periodics for %v", scheduler)
		}
	}

	err := buildKeeperConfig(prowConfig, scheduler.Queries, scheduler.MergeMethod, scheduler.ProtectionPolicy, scheduler.ContextOptions, org, repo)
	if err != nil {
		return errors.Wrapf(err, "building KeeperConfig from %v", scheduler)
	}

	if scheduler.Attachments != nil && len(scheduler.Attachments) > 0 {
		buildPlank(prowConfig, scheduler.Attachments)
	}
	return nil
}

func buildPostsubmits(jobConfig *config.JobConfig, items []*job.Postsubmit, orgName string, repoName string) error {
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
	return nil
}

func buildKeeperConfig(prowConfig *config.ProwConfig, queries []*schedulerapi.Query, mergeMethod *string, protectionPolicy *schedulerapi.ProtectionPolicies, contextOptions *schedulerapi.RepoContextPolicy, orgName string, repoName string) error {
	orgSlashRepo := orgSlashRepo(orgName, repoName)

	if queries != nil && len(queries) > 0 {
		err := buildQuery(&prowConfig.Keeper, queries, orgName, repoName)
		if err != nil {
			return errors.Wrapf(err, "building Query for %s from %v", orgSlashRepo, queries)
		}
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
			err := buildBranchProtection(&prowConfig.BranchProtection, protectionPolicy.ProtectionPolicy,
				orgName, repoName, "")
			if err != nil {
				return errors.WithStack(err)
			}
		}
		for k, v := range protectionPolicy.Items {
			err := buildBranchProtection(&prowConfig.BranchProtection, v, orgName, repoName, k)
			if err != nil {
				return errors.WithStack(err)
			}
		}

	}
	if contextOptions != nil {
		policy := keeper.RepoContextPolicy{}
		err := buildRepoContextPolicy(&policy, contextOptions)
		if err != nil {
			return errors.Wrapf(err, "building RepoContextPolicy from %s", orgSlashRepo)
		}
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
	return nil
}

func buildPresubmits(jobConfig *config.JobConfig, items []*job.Presubmit, orgName string, repoName string) error {
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
	return nil
}

func buildGlobalBranchProtection(answer *branchprotection.Config,
	globalProtectionPolicy *schedulerapi.GlobalProtectionPolicy) error {
	if globalProtectionPolicy.ProtectTested != nil {
		answer.ProtectTested = *globalProtectionPolicy.ProtectTested
	}
	if globalProtectionPolicy.ProtectionPolicy != nil {
		err := buildBranchProtection(answer, globalProtectionPolicy.ProtectionPolicy, "", "", "")
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func buildBranchProtection(answer *branchprotection.Config,
	protectionPolicy *schedulerapi.ProtectionPolicy, orgName string, repoName string, branchName string) error {
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
				err := buildPolicy(&branch.Policy, protectionPolicy)
				if err != nil {
					return errors.Wrapf(err, "building ProtectionPolicy from %v", protectionPolicy)
				}
			} else {
				err := buildPolicy(&repo.Policy, protectionPolicy)
				if err != nil {
					return errors.Wrapf(err, "building ProtectionPolicy from %v", protectionPolicy)
				}
			}
			org.Repos[repoName] = repo
		} else {
			err := buildPolicy(&org.Policy, protectionPolicy)
			if err != nil {
				return errors.Wrapf(err, "building ProtectionPolicy from %v", protectionPolicy)
			}
		}
		answer.Orgs[orgName] = org
	} else {
		err := buildPolicy(&answer.Policy, protectionPolicy)
		if err != nil {
			return errors.Wrapf(err, "building ProtectionPolicy from %v", protectionPolicy)
		}
	}
	return nil
}

func orgSlashRepo(org string, repo string) string {
	if repo == "" {
		return org
	}
	return fmt.Sprintf("%s/%s", org, repo)
}

func buildBase(answer *job.Base, jobBase *job.Base) error {
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
	return nil
}

func buildBrancher(answer *job.Brancher, brancher *job.Brancher) error {
	if len(brancher.SkipBranches) > 0 {
		answer.SkipBranches = brancher.SkipBranches
	}
	if len(brancher.Branches) > 0 {
		answer.Branches = brancher.Branches
	}
	return nil
}

func buildRegexChangeMatacher(answer *job.RegexpChangeMatcher,
	matcher *job.RegexpChangeMatcher) error {
	if len(matcher.RunIfChanged) > 0 {
		answer.RunIfChanged = matcher.RunIfChanged
	}
	return nil
}

func buildPlank(answer *config.ProwConfig, attachments []*schedulerapi.Attachment) {
	for attachmentIndex := range attachments {
		attachment := attachments[attachmentIndex]
		if attachment.Name == "reportTemplate" {
			answer.Plank.ReportTemplateString = attachment.URLs[0]
		}
		/*
			if attachment.Name == "jobURLPrefix" {
				answer.Plank.JobURLPrefix = attachment.URLs[0]
			}
			if attachment.Name == "jobURLTemplate" {
				answer.Plank.JobURLTemplateString = attachment.URLs[0]
			}
		*/
	}
}

func buildPeriodics(answer *config.JobConfig, periodics *schedulerapi.Periodics) error {
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
			if len(schedulerPeriodic.Tags) > 0 {
				periodic.Tags = schedulerPeriodic.Tags
			}
			err := buildBase(&periodic.Base, &schedulerPeriodic.Base)
			if err != nil {
				return errors.Wrapf(err, "building periodic for %v", periodic)
			}
			answer.Periodics = append(answer.Periodics, periodic)
		}
	}
	return nil
}

func buildMerger(answer *keeper.Config, merger *schedulerapi.Merger, org string, repo string) error {
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
		err := buildContextPolicy(&answer.ContextOptions.ContextPolicy, merger.ContextPolicy)
		if err != nil {
			return errors.Wrapf(err, "building ContextPolicy for %v", merger.ContextPolicy)
		}
	}
	return nil
}

func buildRepoContextPolicy(answer *keeper.RepoContextPolicy,
	repoContextPolicy *schedulerapi.RepoContextPolicy) error {
	err := buildContextPolicy(&answer.ContextPolicy, repoContextPolicy.ContextPolicy)
	if err != nil {
		return errors.Wrapf(err, "building ContextPolicy for %v", repoContextPolicy)
	}
	if repoContextPolicy.Branches != nil {
		for branch, policy := range repoContextPolicy.Branches.Items {
			if answer.Branches == nil {
				answer.Branches = make(map[string]keeper.ContextPolicy)
			}
			tidePolicy := keeper.ContextPolicy{}
			err := buildContextPolicy(&tidePolicy, policy)
			if err != nil {
				return errors.Wrapf(err, "building ContextPolicy for %v", policy)
			}
			answer.Branches[branch] = tidePolicy
		}
	}
	return nil
}

func buildContextPolicy(answer *keeper.ContextPolicy,
	contextOptions *schedulerapi.ContextPolicy) error {
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
	return nil
}

func buildQuery(answer *keeper.Config, queries []*schedulerapi.Query, org string, repo string) error {
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
	return nil
}
