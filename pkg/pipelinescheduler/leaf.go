package pipelinescheduler

import (
	schedulerapi "github.com/jenkins-x-plugins/jx-gitops/pkg/apis/scheduler/v1alpha1"
)

// SchedulerLeaf defines a pipeline scheduler leaf
type SchedulerLeaf struct {
	*schedulerapi.SchedulerSpec
	Org  string
	Repo string
}
