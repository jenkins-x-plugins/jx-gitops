package pipelinescheduler

import (
	"github.com/jenkins-x/jx-gitops/pkg/schedulerapi"
)

// SchedulerLeaf defines a pipeline scheduler leaf
type SchedulerLeaf struct {
	*schedulerapi.SchedulerSpec
	Org  string
	Repo string
}
