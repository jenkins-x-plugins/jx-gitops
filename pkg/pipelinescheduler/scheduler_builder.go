package pipelinescheduler

import (
	"strings"

	jenkinsio "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io"
	jenkinsv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-gitops/pkg/schedulerapi"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/jenkins-x/lighthouse-client/pkg/config"
	"github.com/jenkins-x/lighthouse-client/pkg/config/branchprotection"
	"github.com/jenkins-x/lighthouse-client/pkg/config/job"
	"github.com/jenkins-x/lighthouse-client/pkg/config/keeper"
	"github.com/jenkins-x/lighthouse-client/pkg/plugins"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// DefaultAgent is the default agent vaule
	DefaultAgent = "tekton"
	// DefaultMergeType is the default merge type
	DefaultMergeType = "merge"
)

// BuildSchedulers turns prow config in to schedulers
func BuildSchedulers(prowConfig *config.Config, pluginConfig *plugins.Configuration) ([]*jenkinsv1.SourceRepository, map[string]*jenkinsv1.SourceRepository, map[string]*schedulerapi.Scheduler, error) {
	log.Logger().Info("Building scheduler resources from prow config")
	sourceRepos := make(map[string]*jenkinsv1.SourceRepository, 0)
	if prowConfig.Presubmits != nil {
		for repo := range prowConfig.Presubmits {
			orgRepo := strings.Split(repo, "/")
			sourceRepos[repo] = buildSourceRepo(orgRepo[0], orgRepo[1])
		}
	}
	if prowConfig.Postsubmits != nil {
		for repo := range prowConfig.Postsubmits {
			orgRepo := strings.Split(repo, "/")
			if _, ok := sourceRepos[repo]; !ok {
				sourceRepos[repo] = buildSourceRepo(orgRepo[0], orgRepo[1])
			}
		}
	}
	schedulers := make(map[string]*schedulerapi.Scheduler, 0)
	sourceRepoSlice := make([]*jenkinsv1.SourceRepository, 0, len(sourceRepos))
	for sourceRepoName, sourceRepo := range sourceRepos {
		scheduler, err := buildScheduler(sourceRepoName, prowConfig, pluginConfig)
		if err == nil {
			sourceRepo.Spec.Scheduler = jenkinsv1.ResourceReference{
				Name: scheduler.Name,
				Kind: "Scheduler",
			}
			schedulers[scheduler.Name] = scheduler
			sourceRepoSlice = append(sourceRepoSlice, sourceRepo)
		}
	}
	defaultScheduler := buildDefaultScheduler(prowConfig)
	if defaultScheduler != nil {
		schedulers[defaultScheduler.Name] = defaultScheduler
	}
	// TODO Dedupe in to source repo groups
	return sourceRepoSlice, sourceRepos, schedulers, nil
}

func buildSourceRepo(org string, repo string) *jenkinsv1.SourceRepository {
	return &jenkinsv1.SourceRepository{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SourceRepository",
			APIVersion: jenkinsio.GroupName + "/" + jenkinsio.Version,
		},
		Spec: jenkinsv1.SourceRepositorySpec{
			Org:          org,
			Repo:         repo,
			Provider:     giturl.GitHubURL,
			ProviderName: "github",
		},
	}
}

func buildScheduler(repo string, prowConfig *config.Config, pluginConfig *plugins.Configuration) (*schedulerapi.Scheduler, error) {
	scheduler := &schedulerapi.Scheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scheduler",
			APIVersion: jenkinsio.GroupName + "/" + jenkinsio.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.Replace(repo, "/", "-", -1) + "-scheduler",
		},
		Spec: schedulerapi.SchedulerSpec{
			SchedulerAgent:  buildSchedulerAgent(),
			Policy:          buildSchedulerGlobalProtectionPolicy(prowConfig),
			Presubmits:      buildSchedulerPresubmits(repo, prowConfig),
			Postsubmits:     buildSchedulerPostsubmits(repo, prowConfig),
			Trigger:         buildSchedulerTrigger(repo, pluginConfig),
			Approve:         buildSchedulerApprove(repo, pluginConfig),
			LGTM:            buildSchedulerLGTM(repo, pluginConfig),
			ExternalPlugins: buildSchedulerExternalPlugins(repo, pluginConfig),
			Merger:          buildSchedulerMerger(repo, prowConfig),
			Plugins:         buildSchedulerPlugins(repo, pluginConfig),
			ConfigUpdater:   buildSchedulerConfigUpdater(repo, pluginConfig),
			Welcome:         buildSchedulerWelcome(pluginConfig),
		},
	}
	return scheduler, nil
}

func buildDefaultScheduler(prowConfig *config.Config) *schedulerapi.Scheduler {
	scheduler := &schedulerapi.Scheduler{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Scheduler",
			APIVersion: jenkinsio.GroupName + "/" + jenkinsio.Version,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-scheduler",
		},
		Spec: schedulerapi.SchedulerSpec{
			Periodics:   buildSchedulerPeriodics(prowConfig),
			Attachments: buildSchedulerAttachments(prowConfig),
		},
	}
	return scheduler
}

func buildSchedulerAttachments(configuration *config.Config) []*schedulerapi.Attachment {
	attachments := make([]*schedulerapi.Attachment, 0)
	/*
		jobURLPrefix := configuration.Plank.JobURLPrefix
		if jobURLPrefix != "" {
			attachments = buildSchedulerAttachment("jobURLPrefix", jobURLPrefix, attachments)
		}
		jobURLTemplate := configuration.Plank.JobURLTemplateString
		if jobURLTemplate != "" {
			attachments = buildSchedulerAttachment("jobURLTemplate", jobURLTemplate, attachments)
		}
	*/
	reportTemplate := configuration.Plank.ReportTemplateString
	if reportTemplate != "" {
		attachments = buildSchedulerAttachment("reportTemplate", reportTemplate, attachments)
	}
	if len(attachments) > 0 {
		return attachments
	}
	return nil
}

func buildSchedulerAttachment(name string, value string, attachments []*schedulerapi.Attachment) []*schedulerapi.Attachment {
	return append(attachments, &schedulerapi.Attachment{
		Name: name,
		URLs: []string{value},
	})
}

func buildSchedulerPeriodics(configuration *config.Config) *schedulerapi.Periodics {
	periodics := configuration.Periodics
	if periodics != nil && len(periodics) > 0 {

		schedulerPeriodics := &schedulerapi.Periodics{
			Items: make([]*job.Periodic, 0),
		}
		for i := range periodics {
			periodic := periodics[i]
			schedulerPeriodics.Items = append(schedulerPeriodics.Items, &periodic)

		}
		return schedulerPeriodics
	}
	return nil
}

func buildSchedulerWelcome(configuration *plugins.Configuration) []*schedulerapi.Welcome {
	welcomes := configuration.Welcome
	if welcomes != nil && len(welcomes) > 0 {
		schedulerWelcomes := make([]*schedulerapi.Welcome, 0)
		for _, welcome := range welcomes {
			schedulerWelcomes = append(schedulerWelcomes, &schedulerapi.Welcome{MessageTemplate: &welcome.MessageTemplate})

		}
		return schedulerWelcomes
	}
	return nil
}

func buildSchedulerConfigUpdater(repo string, pluginConfig *plugins.Configuration) *schedulerapi.ConfigUpdater {
	if ps, ok := pluginConfig.Plugins[repo]; !ok {
		for _, plugin := range ps {
			if plugin == "config-updater" {
				configMapSpec := make(map[string]schedulerapi.ConfigMapSpec)
				for location, conf := range pluginConfig.ConfigUpdater.Maps {
					spec := schedulerapi.ConfigMapSpec{
						Name:                 conf.Name,
						Namespace:            conf.Namespace,
						Key:                  conf.Key,
						AdditionalNamespaces: conf.AdditionalNamespaces,
						Namespaces:           conf.Namespaces,
					}
					configMapSpec[location] = spec
				}
				return &schedulerapi.ConfigUpdater{
					/* TODO removed
					PluginFile: pluginConfig.ConfigUpdater.PluginFile,
					ConfigFile: pluginConfig.ConfigUpdater.ConfigFile,
					*/
					Map: configMapSpec,
				}
			}
		}
	}
	return nil
}

func buildSchedulerPlugins(repo string, pluginConfig *plugins.Configuration) *schedulerapi.ReplaceableSliceOfStrings {
	if ps, ok := pluginConfig.Plugins[repo]; ok {
		pluginList := &schedulerapi.ReplaceableSliceOfStrings{
			Items: make([]string, 0),
		}
		for _, plugin := range ps {
			pluginList.Items = append(pluginList.Items, plugin)
		}
		if len(pluginList.Items) > 0 {
			return pluginList
		}

	}
	return nil
}

func buildSchedulerMerger(repo string, prowConfig *config.Config) *schedulerapi.Merger {
	tide := prowConfig.Keeper
	merger := &schedulerapi.Merger{
		SyncPeriod:         &tide.SyncPeriod,
		StatusUpdatePeriod: &tide.StatusUpdatePeriod,
		TargetURL:          &tide.TargetURL,
		PRStatusBaseURL:    &tide.PRStatusBaseURL,
		BlockerLabel:       &tide.BlockerLabel,
		SquashLabel:        &tide.BlockerLabel,
		MaxGoroutines:      &tide.MaxGoroutines,
		ContextPolicy:      buildSchedulerContextPolicy(repo, &tide),
	}
	if mergeType, ok := tide.MergeType[repo]; ok {
		mergeTypeStr := string(mergeType)
		merger.MergeType = &mergeTypeStr

	} else {
		defaultMergeType := string(DefaultMergeType)
		merger.MergeType = &defaultMergeType
	}
	return merger
}

func buildSchedulerContextPolicy(orgRepo string, tideConfig *keeper.Config) *schedulerapi.ContextPolicy {
	orgRepoArr := strings.Split(orgRepo, "/")
	orgContextPolicy, orgContextPolicyFound := tideConfig.ContextOptions.Orgs[orgRepoArr[0]]
	if orgContextPolicyFound {
		repoContextPolicy, repoContextPolicyFound := orgContextPolicy.Repos[orgRepoArr[1]]
		if repoContextPolicyFound {
			repoPolicy := schedulerapi.ContextPolicy{}
			repoPolicy.OptionalContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: repoContextPolicy.OptionalContexts}
			repoPolicy.FromBranchProtection = repoContextPolicy.FromBranchProtection
			repoPolicy.RequiredContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: repoContextPolicy.RequiredContexts}
			repoPolicy.RequiredIfPresentContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: repoContextPolicy.RequiredIfPresentContexts}
			repoPolicy.SkipUnknownContexts = repoContextPolicy.SkipUnknownContexts
			return &repoPolicy
		}
		orgPolicy := schedulerapi.ContextPolicy{}
		orgPolicy.OptionalContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: orgContextPolicy.OptionalContexts}
		orgPolicy.FromBranchProtection = orgContextPolicy.FromBranchProtection
		orgPolicy.RequiredContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: orgContextPolicy.RequiredContexts}
		orgPolicy.RequiredIfPresentContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: orgContextPolicy.RequiredIfPresentContexts}
		orgPolicy.SkipUnknownContexts = orgContextPolicy.SkipUnknownContexts
		return &orgPolicy

	}
	contextPolicy := schedulerapi.ContextPolicy{}
	globalContextPolicy := tideConfig.ContextOptions
	contextPolicy.OptionalContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: globalContextPolicy.OptionalContexts}
	contextPolicy.RequiredIfPresentContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: globalContextPolicy.RequiredIfPresentContexts}
	contextPolicy.RequiredContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: globalContextPolicy.RequiredContexts}
	contextPolicy.FromBranchProtection = globalContextPolicy.FromBranchProtection
	contextPolicy.SkipUnknownContexts = globalContextPolicy.SkipUnknownContexts
	return &contextPolicy
}

func buildSchedulerRepoContextPolicy(orgRepo string, tideConfig *keeper.Config) *schedulerapi.RepoContextPolicy {
	orgRepoArr := strings.Split(orgRepo, "/")
	orgContextPolicy, orgContextPolicyFound := tideConfig.ContextOptions.Orgs[orgRepoArr[0]]
	if orgContextPolicyFound {
		repoContextPolicy, repoContextPolicyFound := orgContextPolicy.Repos[orgRepoArr[1]]
		if repoContextPolicyFound && repoContextPolicy.OptionalContexts != nil {
			repoPolicy := schedulerapi.RepoContextPolicy{
				ContextPolicy: &schedulerapi.ContextPolicy{},
			}
			repoPolicy.OptionalContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: repoContextPolicy.OptionalContexts}
			repoPolicy.FromBranchProtection = repoContextPolicy.FromBranchProtection
			repoPolicy.RequiredContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: repoContextPolicy.RequiredContexts}
			repoPolicy.RequiredIfPresentContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: repoContextPolicy.RequiredIfPresentContexts}
			repoPolicy.SkipUnknownContexts = repoContextPolicy.SkipUnknownContexts
			branchPolicies := make(map[string]*schedulerapi.ContextPolicy)
			for branch, policy := range repoContextPolicy.Branches {
				branchPolicy := schedulerapi.ContextPolicy{}
				branchPolicy.OptionalContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: policy.OptionalContexts}
				branchPolicy.FromBranchProtection = policy.FromBranchProtection
				branchPolicy.RequiredContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: policy.RequiredContexts}
				branchPolicy.RequiredIfPresentContexts = &schedulerapi.ReplaceableSliceOfStrings{Items: policy.RequiredIfPresentContexts}
				branchPolicy.SkipUnknownContexts = policy.SkipUnknownContexts
				branchPolicies[branch] = &branchPolicy
			}
			repoPolicy.Branches = &schedulerapi.ReplaceableMapOfStringContextPolicy{Items: branchPolicies}
			return &repoPolicy
		}
	}
	return nil
}

func buildSchedulerQuery(orgRepo string, tideQueries *keeper.Queries) []*schedulerapi.Query {
	queries := make([]*schedulerapi.Query, 0)
	if orgRepo != "" && strings.Contains(orgRepo, "/") {
		for _, tideQuery := range *tideQueries {
			if stringhelpers.StringArrayIndex(tideQuery.Repos, orgRepo) >= 0 {
				query := &schedulerapi.Query{
					ExcludedBranches: &schedulerapi.ReplaceableSliceOfStrings{
						Items: tideQuery.ExcludedBranches,
					},
					IncludedBranches: &schedulerapi.ReplaceableSliceOfStrings{
						Items: tideQuery.IncludedBranches,
					},
					Labels: &schedulerapi.ReplaceableSliceOfStrings{
						Items: tideQuery.Labels,
					},
					MissingLabels: &schedulerapi.ReplaceableSliceOfStrings{
						Items: tideQuery.MissingLabels,
					},
					Milestone:              &tideQuery.Milestone,
					ReviewApprovedRequired: &tideQuery.ReviewApprovedRequired,
				}
				queries = append(queries, query)
			}
		}
	}
	if len(queries) > 0 {
		return queries
	}
	return nil
}

func buildSchedulerExternalPlugins(repo string, pluginConfig *plugins.Configuration) *schedulerapi.ReplaceableSliceOfExternalPlugins {
	pluginList := &schedulerapi.ReplaceableSliceOfExternalPlugins{
		Items: nil,
	}
	if ps, ok := pluginConfig.ExternalPlugins[repo]; ok {
		if ps != nil {
			for _, plugin := range ps {
				if pluginList.Items == nil {
					pluginList.Items = make([]*schedulerapi.ExternalPlugin, 0)
				}
				events := &schedulerapi.ReplaceableSliceOfStrings{
					Items: plugin.Events,
				}
				externalPlugin := &schedulerapi.ExternalPlugin{
					Name:     &plugin.Name,
					Endpoint: &plugin.Endpoint,
					Events:   events,
				}
				pluginList.Items = append(pluginList.Items, externalPlugin)
			}
			return pluginList
		}

	}
	return nil
}

func buildSchedulerLGTM(repo string, pluginConfig *plugins.Configuration) *schedulerapi.Lgtm {
	lgtms := pluginConfig.Lgtm
	for _, lgtm := range lgtms {
		for _, lgtmRepo := range lgtm.Repos {
			if repo == lgtmRepo {
				return &schedulerapi.Lgtm{
					ReviewActsAsLgtm: &lgtm.ReviewActsAsLgtm,
					StoreTreeHash:    &lgtm.StoreTreeHash,
					StickyLgtmTeam:   &lgtm.StickyLgtmTeam,
				}
			}
		}
	}
	return nil
}

func buildSchedulerApprove(repo string, pluginConfig *plugins.Configuration) *schedulerapi.Approve {
	orgRepo := strings.Split(repo, "/")
	approves := pluginConfig.Approve
	for _, approve := range approves {
		for _, approveRepo := range approve.Repos {
			if repo == approveRepo || orgRepo[0] == approveRepo {
				return &schedulerapi.Approve{
					IssueRequired:       &approve.IssueRequired,
					RequireSelfApproval: approve.RequireSelfApproval,
					LgtmActsAsApprove:   &approve.LgtmActsAsApprove,
					IgnoreReviewState:   approve.IgnoreReviewState,
				}
			}
		}
	}
	return nil
}

func buildSchedulerTrigger(repo string, pluginConfig *plugins.Configuration) *schedulerapi.Trigger {
	triggers := pluginConfig.Triggers
	for _, trigger := range triggers {
		for _, triggerRepo := range trigger.Repos {
			if repo == triggerRepo {
				return &schedulerapi.Trigger{
					TrustedOrg:     &trigger.TrustedOrg,
					JoinOrgURL:     &trigger.JoinOrgURL,
					OnlyOrgMembers: &trigger.OnlyOrgMembers,
					IgnoreOkToTest: &trigger.IgnoreOkToTest,
				}
			}
		}
	}
	return nil
}

func buildSchedulerGlobalProtectionPolicy(prowConfig *config.Config) *schedulerapi.GlobalProtectionPolicy {
	return &schedulerapi.GlobalProtectionPolicy{
		ProtectTested: &prowConfig.BranchProtection.ProtectTested,
		ProtectionPolicy: &schedulerapi.ProtectionPolicy{
			Admins:                     prowConfig.BranchProtection.Admins,
			Protect:                    prowConfig.BranchProtection.Protect,
			RequiredPullRequestReviews: buildSchedulerRequiredPullRequestReviews(prowConfig.BranchProtection.RequiredPullRequestReviews),
			RequiredStatusChecks:       buildSchedulerRequiredStatusChecks(prowConfig.BranchProtection.RequiredStatusChecks),
			Restrictions:               buildSchedulerRestrictions(prowConfig.BranchProtection.Restrictions),
		},
	}
}

func buildSchedulerProtectionPolicies(repo string, prowConfig *config.Config) *schedulerapi.ProtectionPolicies {
	orgRepo := strings.Split(repo, "/")
	orgBranchProtection := prowConfig.BranchProtection.GetOrg(orgRepo[0])
	repoBranchProtection := orgBranchProtection.GetRepo(orgRepo[1])
	var protectionPolicies map[string]*schedulerapi.ProtectionPolicy
	for branchName, branch := range repoBranchProtection.Branches {
		if protectionPolicies == nil {
			protectionPolicies = make(map[string]*schedulerapi.ProtectionPolicy)
		}
		protectionPolicies[branchName] = &schedulerapi.ProtectionPolicy{
			Admins:                     branch.Admins,
			Protect:                    branch.Protect,
			RequiredPullRequestReviews: buildSchedulerRequiredPullRequestReviews(branch.RequiredPullRequestReviews),
			RequiredStatusChecks:       buildSchedulerRequiredStatusChecks(branch.RequiredStatusChecks),
			Restrictions:               buildSchedulerRestrictions(branch.Restrictions),
		}
	}
	var repoPolicy *schedulerapi.ProtectionPolicy
	required_pull_request_reviews := buildSchedulerRequiredPullRequestReviews(repoBranchProtection.RequiredPullRequestReviews)
	required_status_checks := buildSchedulerRequiredStatusChecks(repoBranchProtection.RequiredStatusChecks)
	restrictions := buildSchedulerRestrictions(repoBranchProtection.Restrictions)
	if repoBranchProtection.Admins != nil || repoBranchProtection.Protect != nil || required_pull_request_reviews != nil || required_status_checks != nil || restrictions != nil {
		repoPolicy = &schedulerapi.ProtectionPolicy{
			Admins:                     repoBranchProtection.Admins,
			Protect:                    repoBranchProtection.Protect,
			RequiredPullRequestReviews: required_pull_request_reviews,
			RequiredStatusChecks:       required_status_checks,
			Restrictions:               buildSchedulerRestrictions(repoBranchProtection.Restrictions),
		}
	}
	return &schedulerapi.ProtectionPolicies{
		ProtectionPolicy: repoPolicy,
		Items:            protectionPolicies,
	}
}

func buildSchedulerRequiredPullRequestReviews(required_pull_request_reviews *branchprotection.ReviewPolicy) *schedulerapi.ReviewPolicy {
	if required_pull_request_reviews != nil {
		return &schedulerapi.ReviewPolicy{
			DismissalRestrictions: buildSchedulerRestrictions(required_pull_request_reviews.DismissalRestrictions),
			DismissStale:          required_pull_request_reviews.DismissStale,
			RequireOwners:         required_pull_request_reviews.RequireOwners,
			Approvals:             required_pull_request_reviews.Approvals,
		}
	}
	return nil
}

func buildSchedulerRequiredStatusChecks(required_status_checks *branchprotection.ContextPolicy) *schedulerapi.BranchProtectionContextPolicy {
	if required_status_checks != nil {
		return &schedulerapi.BranchProtectionContextPolicy{
			Contexts: &schedulerapi.ReplaceableSliceOfStrings{
				Items: required_status_checks.Contexts,
			},
			Strict: required_status_checks.Strict,
		}
	}
	return nil
}

func buildSchedulerRestrictions(restrictions *branchprotection.Restrictions) *schedulerapi.Restrictions {
	if restrictions != nil {
		return &schedulerapi.Restrictions{
			Users: &schedulerapi.ReplaceableSliceOfStrings{
				Items: restrictions.Users,
			},
			Teams: &schedulerapi.ReplaceableSliceOfStrings{
				Items: restrictions.Teams,
			},
		}
	}
	return nil
}

func buildSchedulerAgent() *schedulerapi.SchedulerAgent {
	defaultAgent := string(DefaultAgent)
	agent := &schedulerapi.SchedulerAgent{
		Agent: &defaultAgent,
	}
	return agent
}

func buildSchedulerPostsubmits(repo string, prowConfig *config.Config) *schedulerapi.Postsubmits {
	schedulerPostsubmits := &schedulerapi.Postsubmits{}
	for postSubmitIndex := range prowConfig.Postsubmits[repo] {
		copy := prowConfig.Postsubmits[repo][postSubmitIndex]
		schedulerPostsubmits.Items = append(schedulerPostsubmits.Items, &copy)
	}
	return schedulerPostsubmits
}

func buildSchedulerPresubmits(repo string, prowConfig *config.Config) *schedulerapi.Presubmits {
	schedulerPresubmits := &schedulerapi.Presubmits{}
	presubmits := prowConfig.Presubmits[repo]
	for presubmitIndex := range presubmits {
		copy := presubmits[presubmitIndex]
		schedulerPresubmits.Items = append(schedulerPresubmits.Items, &copy)
	}
	return schedulerPresubmits
}
