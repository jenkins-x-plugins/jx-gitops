package pipelinescheduler

import (
	"github.com/davecgh/go-spew/spew"
	schedulerapi "github.com/jenkins-x-plugins/jx-gitops/pkg/apis/scheduler/v1alpha1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/lighthouse-client/pkg/config/job"
	"github.com/pkg/errors"
)

//Build combines the slice of schedulers into one, with the most specific schedule config defined last
func Build(schedulers []*schedulerapi.SchedulerSpec) (*schedulerapi.SchedulerSpec, error) {
	var answer *schedulerapi.SchedulerSpec
	for i := len(schedulers) - 1; i >= 0; i-- {
		parent := schedulers[i]
		if answer == nil {
			answer = parent
		} else {
			if answer.SchedulerAgent == nil {
				answer.SchedulerAgent = parent.SchedulerAgent
			} else if parent.SchedulerAgent != nil {
				applyToSchedulerAgent(parent.SchedulerAgent, answer.SchedulerAgent)
			}
			if answer.Policy == nil {
				answer.Policy = parent.Policy
			} else if parent.Policy != nil {
				applyToGlobalProtectionPolicy(parent.Policy, answer.Policy)
			}
			if answer.Presubmits == nil {
				answer.Presubmits = parent.Presubmits
			} else if !answer.Presubmits.Replace && parent.Presubmits != nil {
				err := applyToPreSubmits(parent.Presubmits, answer.Presubmits)
				if err != nil {
					return nil, errors.WithStack(err)
				}
			}
			if answer.Postsubmits == nil {
				answer.Postsubmits = parent.Postsubmits
			} else if !answer.Postsubmits.Replace && parent.Postsubmits != nil {
				err := applyToPostSubmits(parent.Postsubmits, answer.Postsubmits)
				if err != nil {
					return nil, errors.WithStack(err)
				}
			}

			// combine the queries
			if answer.Queries == nil {
				answer.Queries = parent.Queries
			} else {
				applyToQueries(parent.Queries, answer.Queries)
			}
			if answer.MergeMethod == nil {
				answer.MergeMethod = parent.MergeMethod
			}
			if answer.ProtectionPolicy == nil {
				answer.ProtectionPolicy = parent.ProtectionPolicy
			} else if parent.ProtectionPolicy != nil {
				applyToProtectionPolicies(parent.ProtectionPolicy, answer.ProtectionPolicy)
			}
			if answer.ContextOptions == nil {
				answer.ContextOptions = parent.ContextOptions
			} else if parent.ContextOptions != nil {
				applyToRepoContextPolicy(parent.ContextOptions, answer.ContextOptions)
			}

			//TODO: This should probably be an array of triggers, because the plugins yaml is expecting an array
			if answer.Trigger == nil {
				answer.Trigger = parent.Trigger
			} else if parent.Trigger != nil {
				applyToTrigger(parent.Trigger, answer.Trigger)
			}
			if answer.Approve == nil {
				answer.Approve = parent.Approve
			} else if parent.Approve != nil {
				applyToApprove(parent.Approve, answer.Approve)
			}
			if answer.LGTM == nil {
				answer.LGTM = parent.LGTM
			} else if parent.LGTM != nil {
				applyToLgtm(parent.LGTM, answer.LGTM)
			}
			if answer.ExternalPlugins == nil {
				answer.ExternalPlugins = parent.ExternalPlugins
			} else if parent.ExternalPlugins != nil {
				applyToExternalPlugins(parent.ExternalPlugins, answer.ExternalPlugins)
			}
			if answer.Plugins == nil {
				answer.Plugins = parent.Plugins
			} else if parent.Plugins != nil {
				applyToReplaceableSliceOfStrings(parent.Plugins, answer.Plugins)
			}
			if answer.Merger == nil {
				answer.Merger = parent.Merger
			} else if parent.Merger != nil {
				applyToMerger(parent.Merger, answer.Merger)
			}
			if answer.Periodics == nil {
				answer.Periodics = parent.Periodics
			}
			if answer.Attachments == nil {
				answer.Attachments = parent.Attachments
			}
		}
	}
	return answer, nil
}

func applyToTrigger(parent *schedulerapi.Trigger, child *schedulerapi.Trigger) {
	if child.IgnoreOkToTest != nil {
		child.IgnoreOkToTest = parent.IgnoreOkToTest
	}
	if child.JoinOrgURL == nil {
		child.JoinOrgURL = parent.JoinOrgURL
	}
	if child.OnlyOrgMembers == nil {
		child.OnlyOrgMembers = parent.OnlyOrgMembers
	}
	if child.TrustedOrg == nil {
		child.TrustedOrg = parent.TrustedOrg
	}
}

func applyToSchedulerAgent(parent *schedulerapi.SchedulerAgent, child *schedulerapi.SchedulerAgent) {
	if child.Agent == nil {
		child.Agent = parent.Agent
	}
}

func applyToBrancher(parent *job.Brancher, child *job.Brancher) {
	if child.Branches == nil || len(parent.Branches) > 0 {
		child.Branches = parent.Branches
	}
	if child.SkipBranches == nil || len(parent.SkipBranches) > 0 {
		child.SkipBranches = parent.SkipBranches
	}
}

func applyToRegexpChangeMatcher(parent *job.RegexpChangeMatcher, child *job.RegexpChangeMatcher) {
	child.RunIfChanged = parent.RunIfChanged
}

func applyToBase(parent *job.Base, child *job.Base) {
	if child.Name == "" {
		child.Name = parent.Name
	}
	if child.Namespace == nil {
		child.Namespace = parent.Namespace
	}
	if child.Agent == "" {
		child.Agent = parent.Agent
	}
	if child.Cluster == "" {
		child.Cluster = parent.Cluster
	}
	if child.MaxConcurrency <= 0 {
		child.MaxConcurrency = parent.MaxConcurrency
	}
	if child.Labels == nil {
		child.Labels = parent.Labels
	} else if parent.Labels != nil {
		if child.Labels == nil {
			child.Labels = make(map[string]string)
		}
		// Add any labels that are missing
		for pk, pv := range parent.Labels {
			if _, ok := child.Labels[pk]; !ok {
				child.Labels[pk] = pv
			}
		}
	}
}

func applyToMerger(parent *schedulerapi.Merger, child *schedulerapi.Merger) {
	if child.ContextPolicy == nil {
		child.ContextPolicy = parent.ContextPolicy
	} else if parent.ContextPolicy != nil {
		applyToContextPolicy(parent.ContextPolicy, child.ContextPolicy)
	}
	if child.MergeType == nil {
		child.MergeType = parent.MergeType
	}
	if child.MaxGoroutines == nil {
		child.MaxGoroutines = parent.MaxGoroutines
	}
	if child.SquashLabel == nil {
		child.SquashLabel = parent.SquashLabel
	}
	if child.BlockerLabel == nil {
		child.BlockerLabel = parent.BlockerLabel
	}
	if child.PRStatusBaseURL == nil {
		child.PRStatusBaseURL = parent.PRStatusBaseURL
	}
	if child.TargetURL == nil {
		child.TargetURL = parent.TargetURL
	}
	if child.SyncPeriod == nil {
		child.SyncPeriod = parent.SyncPeriod
	}
	if child.StatusUpdatePeriod == nil {
		child.StatusUpdatePeriod = parent.StatusUpdatePeriod
	}
}

func applyToRepoContextPolicy(parent *schedulerapi.RepoContextPolicy, child *schedulerapi.RepoContextPolicy) {
	if child.ContextPolicy == nil {
		child.ContextPolicy = parent.ContextPolicy
	} else if parent.ContextPolicy != nil {
		applyToContextPolicy(parent.ContextPolicy, child.ContextPolicy)
	}
	if child.Branches == nil {
		child.Branches = parent.Branches
	} else if !child.Branches.Replace && parent.Branches != nil {
		if child.Branches.Items == nil {
			child.Branches.Items = make(map[string]*schedulerapi.ContextPolicy)
		}
		for pk, pv := range parent.Branches.Items {
			if cv, ok := child.Branches.Items[pk]; !ok {
				child.Branches.Items[pk] = pv
			} else if pv != nil {
				applyToContextPolicy(pv, cv)
			}
		}
	}
}

func applyToContextPolicy(parent *schedulerapi.ContextPolicy, child *schedulerapi.ContextPolicy) {
	if child.FromBranchProtection == nil {
		child.FromBranchProtection = parent.FromBranchProtection
	}
	if child.SkipUnknownContexts == nil {
		child.SkipUnknownContexts = parent.SkipUnknownContexts
	}
	if child.OptionalContexts == nil {
		child.OptionalContexts = parent.OptionalContexts
	} else if parent.OptionalContexts != nil {
		applyToReplaceableSliceOfStrings(parent.OptionalContexts, child.OptionalContexts)
	}
	if child.RequiredContexts == nil {
		child.RequiredContexts = parent.RequiredContexts
	} else if parent.RequiredContexts != nil {
		applyToReplaceableSliceOfStrings(parent.RequiredContexts, child.RequiredContexts)
	}
	if child.RequiredIfPresentContexts == nil {
		child.RequiredIfPresentContexts = parent.RequiredIfPresentContexts
	} else if parent.RequiredIfPresentContexts != nil {
		applyToReplaceableSliceOfStrings(parent.RequiredIfPresentContexts, child.RequiredIfPresentContexts)
	}
}

func applyToReplaceableSliceOfStrings(parent *schedulerapi.ReplaceableSliceOfStrings, child *schedulerapi.ReplaceableSliceOfStrings) {
	if !child.Replace && parent != nil {
		if child.Items == nil {
			child.Items = make([]string, 0)
		}
		for i := range parent.Items {
			if stringhelpers.StringArrayIndex(child.Items, parent.Items[i]) < 0 {
				child.Items = append(child.Items, parent.Items[i])
			}
		}
	}
}

func applyToLgtm(parent *schedulerapi.Lgtm, child *schedulerapi.Lgtm) {
	if child.StickyLgtmTeam == nil {
		child.StickyLgtmTeam = parent.StickyLgtmTeam
	}
	if child.ReviewActsAsLgtm == nil {
		child.ReviewActsAsLgtm = parent.ReviewActsAsLgtm
	}
	if child.StoreTreeHash == nil {
		child.StoreTreeHash = parent.StoreTreeHash
	}
}

func applyToExternalPlugins(parent *schedulerapi.ReplaceableSliceOfExternalPlugins, child *schedulerapi.ReplaceableSliceOfExternalPlugins) {
	if child.Items == nil {
		child.Items = parent.Items
	} else if !child.Replace {
		child.Items = append(child.Items, parent.Items...)
	}
}

// TODO use this
//func applyToExternalPlugin(parent *schedulerapi.ExternalPlugin, child *schedulerapi.ExternalPlugin) {
//	if child.Name == nil {
//		child.Name = parent.Name
//	}
//	if child.Endpoint == nil {
//		child.Endpoint = parent.Endpoint
//	}
//	if child.Events == nil {
//		child.Events = parent.Events
//	} else if parent.Events != nil {
//		applyToReplaceableSliceOfStrings(parent.Events, child.Events)
//	}
//}

func applyToApprove(parent *schedulerapi.Approve, child *schedulerapi.Approve) {
	if child.IgnoreReviewState == nil {
		child.IgnoreReviewState = parent.IgnoreReviewState
	}
	if child.IssueRequired == nil {
		child.IssueRequired = parent.IssueRequired
	}
	if child.LgtmActsAsApprove == nil {
		child.LgtmActsAsApprove = parent.LgtmActsAsApprove
	}
	if child.RequireSelfApproval == nil {
		child.RequireSelfApproval = parent.RequireSelfApproval
	}
}

func applyToGlobalProtectionPolicy(parent *schedulerapi.GlobalProtectionPolicy, child *schedulerapi.GlobalProtectionPolicy) {
	if child.ProtectionPolicy == nil {
		child.ProtectionPolicy = parent.ProtectionPolicy
	} else if parent.ProtectionPolicy != nil {
		applyToProtectionPolicy(parent.ProtectionPolicy, child.ProtectionPolicy)
	}
	if child.ProtectTested == nil {
		child.ProtectTested = parent.ProtectTested
	}
}

func applyToProtectionPolicy(parent *schedulerapi.ProtectionPolicy, child *schedulerapi.ProtectionPolicy) {
	if child.Protect == nil {
		child.Protect = parent.Protect
	}
	if child.Admins == nil {
		child.Admins = parent.Admins
	}
	if child.Restrictions == nil {
		child.Restrictions = parent.Restrictions
	} else if parent.Restrictions != nil {
		applyToRestrictions(parent.Restrictions, child.Restrictions)
	}
	if child.RequiredPullRequestReviews == nil {
		child.RequiredPullRequestReviews = parent.RequiredPullRequestReviews
	} else if parent.RequiredPullRequestReviews != nil {
		applyToRequiredPullRequestReviews(parent.RequiredPullRequestReviews, child.RequiredPullRequestReviews)
	}
}

func applyToRequiredPullRequestReviews(parent *schedulerapi.ReviewPolicy, child *schedulerapi.ReviewPolicy) {
	if child.Approvals == nil {
		child.Approvals = parent.Approvals
	}
	if child.DismissStale == nil {
		child.DismissStale = parent.DismissStale
	}
	if child.RequireOwners == nil {
		child.RequireOwners = parent.RequireOwners
	}
	if child.DismissalRestrictions == nil {
		child.DismissalRestrictions = parent.DismissalRestrictions
	} else if parent.DismissalRestrictions != nil {
		applyToRestrictions(parent.DismissalRestrictions, child.DismissalRestrictions)
	}
}

func applyToRestrictions(parent *schedulerapi.Restrictions, child *schedulerapi.Restrictions) {
	if child.Teams == nil {
		child.Teams = parent.Teams
	} else if parent.Teams != nil {
		applyToReplaceableSliceOfStrings(parent.Teams, child.Teams)
	}
	if child.Users == nil {
		child.Users = parent.Users
	} else if parent.Users != nil {
		applyToReplaceableSliceOfStrings(parent.Users, child.Users)
	}
}

func applyToPostSubmits(parentPostsubmits *schedulerapi.Postsubmits, childPostsubmits *schedulerapi.Postsubmits) error {
	if childPostsubmits.Items == nil {
		childPostsubmits.Items = make([]*job.Postsubmit, 0)
	}
	// Work through each of the post submits in the parent. If we can find a name based match in child,
	// we apply it to the child, otherwise we append it
	for _, parent := range parentPostsubmits.Items {
		var found []*job.Postsubmit
		for _, postsubmit := range childPostsubmits.Items {
			if postsubmit.Name != "" && parent.Name != "" && postsubmit.Name == parent.Name {
				found = append(found, postsubmit)
			}
		}
		if len(found) > 1 {
			return errors.Errorf("more than one postsubmit with name %v in %s", parent.Name, spew.Sdump(childPostsubmits))
		} else if len(found) == 1 {
			child := found[0]
			*child = *parent
		} else {
			childPostsubmits.Items = append(childPostsubmits.Items, parent)
		}
	}
	return nil
}

func applyToPreSubmits(parentPresubmits *schedulerapi.Presubmits, childPresubmits *schedulerapi.Presubmits) error {
	if childPresubmits.Items == nil {
		childPresubmits.Items = make([]*job.Presubmit, 0)
	}
	// Work through each of the presubmits in the parent. If we can find a name based match in child,
	// we apply it to the child, otherwise we append it
	for _, parent := range parentPresubmits.Items {
		var found []*job.Presubmit
		for _, child := range childPresubmits.Items {
			if child.Name == parent.Name {
				found = append(found, child)
			}
		}
		if len(found) > 1 {
			return errors.Errorf("more than one presubmit with name %v in %s", parent.Name, spew.Sdump(parentPresubmits))
		} else if len(found) == 1 {
			child := found[0]
			*child = *parent
		} else {
			childPresubmits.Items = append(childPresubmits.Items, parent)
		}
	}
	return nil
}

func applyToProtectionPolicies(parent *schedulerapi.ProtectionPolicies,
	child *schedulerapi.ProtectionPolicies) {
	if child.ProtectionPolicy == nil {
		child.ProtectionPolicy = parent.ProtectionPolicy
	} else if parent.ProtectionPolicy != nil {
		applyToProtectionPolicy(parent.ProtectionPolicy, child.ProtectionPolicy)
	}
	if child.Items == nil {
		child.Items = parent.Items
	} else if !child.Replace {
		for k, v := range parent.Items {
			if _, ok := child.Items[k]; !ok {
				child.Items[k] = v
			}
		}
	}
}

func applyToQueries(parents []*schedulerapi.Query, children []*schedulerapi.Query) {
	for _, v := range parents {
		children = append(children, v)
	}
}
