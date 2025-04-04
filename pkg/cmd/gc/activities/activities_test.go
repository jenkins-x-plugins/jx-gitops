package activities

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/lighthouse-client/pkg/apis/lighthouse/v1alpha1"
	fakelh "github.com/jenkins-x/lighthouse-client/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	fakedyn "k8s.io/client-go/dynamic/fake"
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
				Labels:    createLabels("PR-1", "3"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
				Status:             v1.ActivityStatusTypeFailed,
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
				Status:   v1.ActivityStatusTypeRunning,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "3",
				Namespace: ns,
				Labels:    createLabels("PR-1", "1"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThirtyOneDays},
				Status:             v1.ActivityStatusTypeSucceeded,
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
				Status:             v1.ActivityStatusTypeAborted,
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
				Pipeline:         "org/project/PR-1",
				StartedTimestamp: &metav1.Time{Time: nowMinusThirtyOneDays},
				Status:           v1.ActivityStatusTypeCancelled,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "5",
				Namespace: ns,
				Labels:    createLabels("batch", "5"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:         "org/project/batch",
				StartedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
				Status:           v1.ActivityStatusTypeTimedOut,
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "6",
				Namespace: ns,
				Labels:    createLabels("master", "6"),
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:         "org/project/master",
				StartedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
				Status:           v1.ActivityStatusTypeError,
			},
		},
	}

	paRuntimes := PipelineActivitiesToRuntimes(pas)
	jxClient := jxfake.NewSimpleClientset(paRuntimes...)

	lhJobs := ToLighthouseJobs(pas)
	lhjRuntimes := LighthouseJobsToRuntimes(lhJobs)
	lhClient := fakelh.NewSimpleClientset(lhjRuntimes...)

	scheme := runtime.NewScheme()
	tknClient := fakedyn.NewSimpleDynamicClientWithCustomListKinds(scheme, map[schema.GroupVersionResource]string{PipelineResource: "PipelineRunList"})

	tknPipelineRuns := ToPipelineRuns(t, pas)
	for i := range tknPipelineRuns {
		err := tknClient.Tracker().Create(PipelineResource, tknPipelineRuns[i], ns)
		assert.NoError(t, err)
	}

	_, o := NewCmdGCActivities()
	o.Namespace = ns
	o.JXClient = jxClient
	o.LHClient = lhClient
	o.DynamicClient = tknClient

	lhjobs, err := lhClient.LighthouseV1alpha1().LighthouseJobs(ns).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	t.Logf("has %d LighthouseJobs\n", len(lhjobs.Items))

	prRuns, err := tknClient.Resource(PipelineResource).Namespace(ns).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)
	t.Logf("has %d PipelineRuns\n", len(prRuns.Items))

	// Delete a pipeline run to ensure that gc activites don't try to delete something that does not exist.
	err = tknClient.Resource(PipelineResource).Namespace(ns).Delete(ctx, prRuns.Items[0].GetName(), metav1.DeleteOptions{})
	assert.NoError(t, err)

	err = o.Run()
	assert.NoError(t, err)

	activityList, err := jxClient.JenkinsV1().PipelineActivities(ns).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)

	assert.Len(t, activityList.Items, 3, "Two of the activities should've been garbage collected")

	var verifier []bool
	for _, v := range activityList.Items {
		if v.BranchName() == "batch" || v.BranchName() == "PR-1" {
			verifier = append(verifier, true)
		}
	}
	assert.Len(t, verifier, 2, "Both PR and Batch builds should've been garbage collected")

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

func ToPipelineRuns(t *testing.T, list []*v1.PipelineActivity) []runtime.Object {
	var answer []runtime.Object
	for _, r := range list {
		j := &unstructured.Unstructured{}
		j.SetName(r.ObjectMeta.Name)
		j.SetNamespace(r.ObjectMeta.Namespace)
		j.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   PipelineResource.Group,
			Version: PipelineResource.Version,
			Kind:    PipelineResource.Resource,
		})

		answer = append(answer, j)
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
		PrLabel:                                 buildNum,
	}
}
