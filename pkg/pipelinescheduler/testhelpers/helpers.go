package testhelpers

import (
	"io/ioutil"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	schedulerapi "github.com/jenkins-x-plugins/jx-gitops/pkg/apis/scheduler/v1alpha1"
	"github.com/jenkins-x/lighthouse-client/pkg/config/job"

	"github.com/ghodss/yaml"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/pipelinescheduler"
	"github.com/jenkins-x/lighthouse-client/pkg/config"
	"github.com/jenkins-x/lighthouse-client/pkg/plugins"

	"github.com/stretchr/testify/assert"

	"github.com/pborman/uuid"
)

// CompleteScheduler returns a SchedulerSpec completely filled with dummy data
func CompleteScheduler() *schedulerapi.SchedulerSpec {
	return &schedulerapi.SchedulerSpec{
		Policy: pointerToGlobalProtectionPolicy(),
		Merger: &schedulerapi.Merger{
			ContextPolicy: &schedulerapi.ContextPolicy{
				OptionalContexts:          PointerToReplaceableSliceOfStrings(),
				RequiredContexts:          PointerToReplaceableSliceOfStrings(),
				RequiredIfPresentContexts: PointerToReplaceableSliceOfStrings(),
			},
			MergeType:          pointerToUUID(),
			TargetURL:          pointerToUUID(),
			PRStatusBaseURL:    pointerToUUID(),
			BlockerLabel:       pointerToUUID(),
			SquashLabel:        pointerToUUID(),
			MaxGoroutines:      pointerToRandomNumber(),
			StatusUpdatePeriod: pointerToRandomDuration(),
			SyncPeriod:         pointerToRandomDuration(),
		},
		Presubmits: &schedulerapi.Presubmits{
			Items: []*job.Presubmit{
				{
					Base: job.Base{
						Name: "cheese",
					},
				},
			},
		},
		Postsubmits: &schedulerapi.Postsubmits{
			Items: []*job.Postsubmit{
				{
					Base: job.Base{
						Name: "cheese",
					},
				},
			},
		},
		Trigger: &schedulerapi.Trigger{
			IgnoreOkToTest: pointerToTrue(),
			JoinOrgURL:     pointerToUUID(),
			OnlyOrgMembers: pointerToTrue(),
			TrustedOrg:     pointerToUUID(),
		},
		SchedulerAgent: &schedulerapi.SchedulerAgent{
			Agent: pointerToUUID(),
		},
		Approve: &schedulerapi.Approve{
			RequireSelfApproval: pointerToTrue(),
			LgtmActsAsApprove:   pointerToTrue(),
			IssueRequired:       pointerToTrue(),
			IgnoreReviewState:   pointerToTrue(),
		},
		ExternalPlugins: &schedulerapi.ReplaceableSliceOfExternalPlugins{
			Items: []*schedulerapi.ExternalPlugin{
				{
					Name:     pointerToUUID(),
					Events:   PointerToReplaceableSliceOfStrings(),
					Endpoint: pointerToUUID(),
				},
			},
		},
		LGTM: &schedulerapi.Lgtm{
			StoreTreeHash:    pointerToTrue(),
			ReviewActsAsLgtm: pointerToTrue(),
			StickyLgtmTeam:   pointerToUUID(),
		},
		Plugins: PointerToReplaceableSliceOfStrings(),
	}
}

func pointerToTrue() *bool {
	b := true
	return &b
}

func pointerToUUID() *string {
	s := uuid.New()
	return &s
}

func pointerToRandomNumber() *int {
	i := rand.Int() // #nosec
	return &i
}

func pointerToRandomDuration() *time.Duration {
	i := rand.Int63()
	duration := time.Duration(i)
	return &duration
}

// PointerToReplaceableSliceOfStrings creaters a ReplaceableSliceOfStrings and returns its pointer
func PointerToReplaceableSliceOfStrings() *schedulerapi.ReplaceableSliceOfStrings {
	return &schedulerapi.ReplaceableSliceOfStrings{
		Items: []string{
			uuid.New(),
		},
	}
}

func pointerToContextPolicy() *schedulerapi.ContextPolicy {
	return &schedulerapi.ContextPolicy{
		SkipUnknownContexts:       pointerToTrue(),
		FromBranchProtection:      pointerToTrue(),
		RequiredIfPresentContexts: PointerToReplaceableSliceOfStrings(),
		RequiredContexts:          PointerToReplaceableSliceOfStrings(),
		OptionalContexts:          PointerToReplaceableSliceOfStrings(),
	}
}

func pointerToGlobalProtectionPolicy() *schedulerapi.GlobalProtectionPolicy {
	return &schedulerapi.GlobalProtectionPolicy{
		ProtectTested:    pointerToTrue(),
		ProtectionPolicy: pointerToProtectionPolicy(),
	}
}

func pointerToProtectionPolicy() *schedulerapi.ProtectionPolicy {
	return &schedulerapi.ProtectionPolicy{
		Restrictions: &schedulerapi.Restrictions{
			Users: PointerToReplaceableSliceOfStrings(),
			Teams: PointerToReplaceableSliceOfStrings(),
		},
		Admins: pointerToTrue(),
		RequiredPullRequestReviews: &schedulerapi.ReviewPolicy{
			DismissalRestrictions: &schedulerapi.Restrictions{
				Users: PointerToReplaceableSliceOfStrings(),
				Teams: PointerToReplaceableSliceOfStrings(),
			},
		},
		RequiredStatusChecks: &schedulerapi.BranchProtectionContextPolicy{
			Strict:   pointerToTrue(),
			Contexts: PointerToReplaceableSliceOfStrings(),
		},
		Protect: pointerToTrue(),
	}
}

// SchedulerFile contains a list of leaf files to build the scheduler from
type SchedulerFile struct {
	// Filenames is the hierarchy with the leaf at the right
	Filenames []string
	Org       string
	Repo      string
}

// BuildAndValidateProwConfig takes a list of schedulerFiles and builds them to a Prow config,
// and validates them against the expectedConfigFilename and expectedPluginsFilename that make up the prow config.
// Filepaths are relative to the baseDir
func BuildAndValidateProwConfig(t *testing.T, baseDir string, expectedConfigFilename string,
	expectedPluginsFilename string, schedulerFiles []SchedulerFile) {
	var expectedConfig config.Config
	if expectedConfigFilename != "" {
		cfgBytes, err := ioutil.ReadFile(filepath.Join(baseDir, expectedConfigFilename))
		assert.NoError(t, err)
		err = yaml.Unmarshal(cfgBytes, &expectedConfig)
		assert.NoError(t, err)
	}

	var expectedPlugins plugins.Configuration
	if expectedPluginsFilename != "" {
		bytes, err := ioutil.ReadFile(filepath.Join(baseDir, expectedPluginsFilename))
		assert.NoError(t, err)
		err = yaml.Unmarshal(bytes, &expectedPlugins)
		assert.NoError(t, err)
	}

	schedulerLeaves := make([]*pipelinescheduler.SchedulerLeaf, 0)
	for _, sfs := range schedulerFiles {
		schedulers := make([]*schedulerapi.SchedulerSpec, 0)
		for _, f := range sfs.Filenames {
			bytes, err := ioutil.ReadFile(filepath.Join(baseDir, f))
			assert.NoError(t, err)
			s := schedulerapi.SchedulerSpec{}
			err = yaml.Unmarshal(bytes, &s)
			assert.NoError(t, err)
			schedulers = append(schedulers, &s)
		}
		s, err := pipelinescheduler.Build(schedulers)
		assert.NoError(t, err)
		schedulerLeaves = append(schedulerLeaves, &pipelinescheduler.SchedulerLeaf{
			Repo:          sfs.Repo,
			Org:           sfs.Org,
			SchedulerSpec: s,
		})
	}

	cfg, plugs, err := pipelinescheduler.BuildProwConfig(schedulerLeaves)
	assert.NoError(t, err)
	if expectedConfigFilename != "" {
		expected, err := yaml.Marshal(&expectedConfig)
		assert.NoError(t, err)
		actual, err := yaml.Marshal(cfg)
		assert.NoError(t, err)
		assert.NotNil(t, expected)
		if !assert.Equal(t, string(expected), string(actual)) {
			t.Logf("config expected: %s\n", string(expected))
			t.Logf("got: %s\n", string(actual))
		}
	}
	if expectedPluginsFilename != "" {
		expected, err := yaml.Marshal(&expectedPlugins)
		assert.NoError(t, err)
		actual, err := yaml.Marshal(plugs)
		assert.NoError(t, err)
		if !assert.Equal(t, string(expected), string(actual)) {
			t.Logf("plugins expected: %s\n", string(expected))
			t.Logf("got: %s\n", string(actual))

		}
	}
}
