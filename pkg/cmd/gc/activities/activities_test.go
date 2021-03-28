// +build unit

package activities_test

import (
	"context"
	"testing"
	"time"

	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/gc/activities"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGCPipelineActivities(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	ns := "jx"
	nowMinusThirtyOneDays := time.Now().AddDate(0, 0, -31)
	nowMinusThreeDays := time.Now().AddDate(0, 0, -3)
	nowMinusTwoDays := time.Now().AddDate(0, 0, -2)
	nowMinusOneDay := time.Now().AddDate(0, 0, -1)

	jxClient := jxfake.NewSimpleClientset(
		&v1.PipelineActivity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "1",
				Namespace: ns,
				Labels: map[string]string{
					v1.LabelBranch: "PR-1",
				},
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
			},
		},
		&v1.PipelineActivity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "2",
				Namespace: ns,
				Labels: map[string]string{
					v1.LabelBranch: "PR-1",
				},
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline: "org/project/PR-1",
				// No completion time, to make sure this doesn't get deleted.
			},
		},
		&v1.PipelineActivity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "3",
				Namespace: ns,
				Labels: map[string]string{
					v1.LabelBranch: "PR-1",
				},
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusTwoDays},
			},
		},
		&v1.PipelineActivity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "4",
				Namespace: ns,
				Labels: map[string]string{
					v1.LabelBranch: "PR-1",
				},
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusOneDay},
			},
		},

		// To handle potential weirdness around ordering, make sure that the oldest PR activity is in a random
		// spot in the order.
		&v1.PipelineActivity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "0",
				Namespace: ns,
				Labels: map[string]string{
					v1.LabelBranch: "PR-1",
				},
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThirtyOneDays},
			},
		},
		&v1.PipelineActivity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "5",
				Namespace: ns,
				Labels: map[string]string{
					v1.LabelBranch: "batch",
				},
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/batch",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
			},
		},
		&v1.PipelineActivity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "6",
				Namespace: ns,
				Labels: map[string]string{
					v1.LabelBranch: "master",
				},
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/master",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
			},
		},
	)

	_, o := activities.NewCmdGCActivities()
	o.Namespace = ns
	o.JXClient = jxClient

	err := o.Run()
	assert.NoError(t, err)

	activities, err := jxClient.JenkinsV1().PipelineActivities(ns).List(ctx, metav1.ListOptions{})
	assert.NoError(t, err)

	assert.Len(t, activities.Items, 4, "Two of the activities should've been garbage collected")

	var verifier []bool
	for _, v := range activities.Items {
		if v.BranchName() == "batch" || v.BranchName() == "PR-1" {
			verifier = append(verifier, true)
		}
	}
	assert.Len(t, verifier, 3, "Both PR and Batch builds should've been verified")
}
