// +build unit

package activities_test

import (
	"context"
	"testing"
	"time"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/gc/activities"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/lighthouse-client/pkg/apis/lighthouse/v1alpha1"
	fakelh "github.com/jenkins-x/lighthouse-client/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	faketekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestGCPipelineActivities(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	ns := "jx"
	nowMinusThirtyOneDays := time.Now().AddDate(0, 0, -31)
	nowMinusThreeDays := time.Now().AddDate(0, 0, -3)
	nowMinusOneDay := time.Now().AddDate(0, 0, -1)

	pas := []*v1.PipelineActivity{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "1",
				Namespace: ns,
				Labels:    createLabels("PR-1", "1"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "2",
				Namespace: ns,
				Labels:    createLabels("PR-1", "2"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline: "org/project/PR-1",
				// No completion time, to make sure this doesn't get deleted.
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "3",
				Namespace: ns,
				Labels:    createLabels("PR-1", "3"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "4",
				Namespace: ns,
				Labels:    createLabels("PR-1", "4"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusOneDay},
			},
		},

		// To handle potential weirdness around ordering, make sure that the oldest PR activity is in a random
		// spot in the order.
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "0",
				Namespace: ns,
				Labels:    createLabels("PR-1", "0"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThirtyOneDays},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "5",
				Namespace: ns,
				Labels:    createLabels("batch", "5"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/batch",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "6",
				Namespace: ns,
				Labels:    createLabels("master", "6"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/master",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
			},
		},
	}

	paRuntimes := PipelineActivitiesToRuntimes(pas)
	jxClient := jxfake.NewSimpleClientset(paRuntimes...)

	lhJobs := ToLighthouseJobs(pas)
	lhjRuntimes := LighthouseJobsToRuntimes(lhJobs)
	lhClient := fakelh.NewSimpleClientset(lhjRuntimes...)

	tknPipelineRuns := ToPipelineRuns(pas)
	tknRuntimes := PipelineRunsToRuntimes(tknPipelineRuns)
	tknClient := faketekton.NewSimpleClientset(tknRuntimes...)

	_, o := activities.NewCmdGCActivities()
	o.Namespace = ns
	o.JXClient = jxClient
	o.LHClient = lhClient
	o.TknClient = tknClient

	lhjobs, err := lhClient.LighthouseV1alpha1().LighthouseJobs(ns).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	t.Logf("has %d LighthouseJobs\n", len(lhjobs.Items))

	prRuns, err := tknClient.TektonV1beta1().PipelineRuns(ns).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	t.Logf("has %d PipelineRuns\n", len(prRuns.Items))

	// Delete a pipeline run to ensure that gc activites don't try to delete something that does not exist.
	err = tknClient.TektonV1beta1().PipelineRuns(ns).Delete(ctx, prRuns.Items[0].Name, metav1.DeleteOptions{})
	assert.NoError(t, err)

	err = o.Run()
	assert.NoError(t, err)

	activities, err := jxClient.JenkinsV1().PipelineActivities(ns).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)

	assert.Len(t, activities.Items, 3, "Two of the activities should've been garbage collected")

	var verifier []bool
	for _, v := range activities.Items {
		if v.BranchName() == "batch" || v.BranchName() == "PR-1" {
			verifier = append(verifier, true)
		}
	}
	assert.Len(t, verifier, 2, "Both PR and Batch builds should've been verified")

	// lets verify number of LH jobs left
	lhjobs, err = lhClient.LighthouseV1alpha1().LighthouseJobs(ns).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	t.Logf("has %d LighthouseJobs\n", len(lhjobs.Items))
	assert.Len(t, lhjobs.Items, 3, "Number of renaming LighthouseJobs")
}

func PipelineActivitiesToRuntimes(list []*v1.PipelineActivity) []runtime.Object {
	var answer []runtime.Object
	for _, r := range list {
		answer = append(answer, r)
	}
	return answer
}

func LighthouseJobsToRuntimes(list []*v1alpha1.LighthouseJob) []runtime.Object {
	var answer []runtime.Object
	for _, r := range list {
		answer = append(answer, r)
	}
	return answer
}

func ToLighthouseJobs(list []*v1.PipelineActivity) []*v1alpha1.LighthouseJob {
	var answer []*v1alpha1.LighthouseJob
	for _, r := range list {
		j := &v1alpha1.LighthouseJob{
			ObjectMeta: r.ObjectMeta,
		}
		answer = append(answer, j)
	}
	return answer
}

func ToPipelineRuns(list []*v1.PipelineActivity) []*v1beta1.PipelineRun {
	var answer []*v1beta1.PipelineRun
	for _, r := range list {
		j := &v1beta1.PipelineRun{
			ObjectMeta: r.ObjectMeta,
		}
		answer = append(answer, j)
	}
	return answer
}

func PipelineRunsToRuntimes(list []*v1beta1.PipelineRun) []runtime.Object {
	var answer []runtime.Object
	for _, r := range list {
		answer = append(answer, r)
	}
	return answer
}

func createLabels(branch, buildNum string) map[string]string {
	t := "postsubmit"
	if branch != "master" && branch != "main" {
		t = "presubmit"
	}
	return map[string]string{
		"lighthouse.jenkins-x.io/baseSHA":       "8f17a6629f58bf7e7d6de59c6d429c081ac3d396",
		"lighthouse.jenkins-x.io/branch":        branch,
		"lighthouse.jenkins-x.io/buildNum":      buildNum,
		"lighthouse.jenkins-x.io/context":       "mycontext",
		"lighthouse.jenkins-x.io/job":           "mycontext",
		"lighthouse.jenkins-x.io/lastCommitSHA": "mysha",
		"lighthouse.jenkins-x.io/refs.org":      "org",
		"lighthouse.jenkins-x.io/refs.repo":     "project",
		"lighthouse.jenkins-x.io/type":          t,
		activities.PrLabel:                      buildNum,
	}
}
