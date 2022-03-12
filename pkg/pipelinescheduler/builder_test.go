//go:build unit
// +build unit

package pipelinescheduler_test

import (
	"testing"

	schedulerapi "github.com/jenkins-x-plugins/jx-gitops/pkg/apis/scheduler/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/pipelinescheduler"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/pipelinescheduler/testhelpers"
	"github.com/stretchr/testify/assert"
)

func TestBuildWithEverythingInParent(t *testing.T) {
	child := &schedulerapi.SchedulerSpec{
		// Override nothing, everything comes from
	}
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*schedulerapi.SchedulerSpec{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent, merged)
}

func TestBuildWithEverythingInChild(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*schedulerapi.SchedulerSpec{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, child, merged)
}

func TestBuildWithSomePropertiesMergedLgtm(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	child.LGTM.ReviewActsAsLgtm = nil
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*schedulerapi.SchedulerSpec{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent.LGTM.ReviewActsAsLgtm, merged.LGTM.ReviewActsAsLgtm)
	assert.Equal(t, child.LGTM.StickyLgtmTeam, merged.LGTM.StickyLgtmTeam)
}

func TestBuildWithLgtmEmptyInChild(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	child.LGTM = &schedulerapi.Lgtm{}
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*schedulerapi.SchedulerSpec{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, child.LGTM, merged.LGTM)
}

func TestBuildWithSomePropertiesMergedMerger(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	child.Merger.ContextPolicy = nil
	child.Merger.MergeType = nil
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*schedulerapi.SchedulerSpec{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent.Merger.ContextPolicy, merged.Merger.ContextPolicy)
	assert.Equal(t, parent.Merger.MergeType, merged.Merger.MergeType)
	assert.Equal(t, child.Merger.SquashLabel, merged.Merger.SquashLabel)
}

func TestBuildWithEmptyMerger(t *testing.T) {
	t.Parallel()
	child := testhelpers.CompleteScheduler()
	child.Merger = &schedulerapi.Merger{}
	parent := testhelpers.CompleteScheduler()
	merged, err := pipelinescheduler.Build([]*schedulerapi.SchedulerSpec{parent, child})
	assert.NoError(t, err)
	assert.Equal(t, parent.Merger, merged.Merger)
}
