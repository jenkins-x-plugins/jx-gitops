// +build unit

package activities_test

import (
	"context"
	"testing"
	"time"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/gc/activities"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	faketekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGCPipelineActivities(t *testing.T) {
	t.Parallel()

	ctx := context.TODO()
	ns := "jx"
	prName := "pr-z9wm6"
	nowMinusThirtyOneDays := time.Now().AddDate(0, 0, -31)
	nowMinusThreeDays := time.Now().AddDate(0, 0, -3)
	nowMinusOneDay := time.Now().AddDate(0, 0, -1)

	tknClient  := faketekton.NewSimpleClientset(
		&v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:                       prName+"0",
				Namespace:                  ns,
			},
		},
		&v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:                       prName+"1",
				Namespace:                  ns,
			},
		},
		&v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:                       prName+"2",
				Namespace:                  ns,
			},
		},
		&v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:                       prName+"3",
				Namespace:                  ns,
			},
		},
		&v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:                       prName+"4",
				Namespace:                  ns,
			},
		},
		&v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:                       prName+"5",
				Namespace:                  ns,
			},
		},
		&v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:                       prName+"6",
				Namespace:                  ns,
			},
		},
	)

	jxClient := jxfake.NewSimpleClientset(
		&v1.PipelineActivity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "1",
				Namespace: ns,
				Labels: map[string]string{
					v1.LabelBranch: "PR-1",
					activities.PrLabel: prName+"0",
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
					activities.PrLabel: prName+"1",
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
					activities.PrLabel: prName+"2",
				},
			},
			Spec: v1.PipelineActivitySpec{
				Pipeline:           "org/project/PR-1",
				CompletedTimestamp: &metav1.Time{Time: nowMinusThreeDays},
			},
		},
		&v1.PipelineActivity{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "4",
				Namespace: ns,
				Labels: map[string]string{
					v1.LabelBranch: "PR-1",
					activities.PrLabel: prName+"3",
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
					activities.PrLabel: prName+"4",
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
					activities.PrLabel: prName+"5",
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
					activities.PrLabel: prName+"6",
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
	o.TknClient = tknClient

	err := o.Run()
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
}
