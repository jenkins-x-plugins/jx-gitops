package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestGCJobsDeletesPodsWithJob(t *testing.T) {
	t.Parallel()

	ns := "jx-git-operator"
	oldStart := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	newStart := metav1.NewTime(time.Now().Add(-10 * time.Minute))

	kubeClient := fake.NewClientset(
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "old", Namespace: ns},
			Status:     batchv1.JobStatus{StartTime: &oldStart, Succeeded: 1},
		},
		&batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{Name: "new", Namespace: ns},
			Status:     batchv1.JobStatus{StartTime: &newStart, Succeeded: 1},
		},
	)

	o := &Options{
		Namespace:  ns,
		Age:        time.Hour,
		Keep:       0,
		KubeClient: kubeClient,
	}
	require.NoError(t, o.Run())

	var deletes []k8stesting.DeleteActionImpl
	for _, action := range kubeClient.Actions() {
		if d, ok := action.(k8stesting.DeleteActionImpl); ok {
			deletes = append(deletes, d)
		}
	}
	require.Len(t, deletes, 1)
	assert.Equal(t, "old", deletes[0].GetName())
	require.NotNil(t, deletes[0].DeleteOptions.PropagationPolicy, "a Job deleted without a propagation policy leaves its Pods behind")
	assert.Equal(t, metav1.DeletePropagationBackground, *deletes[0].DeleteOptions.PropagationPolicy)

	remaining, err := kubeClient.BatchV1().Jobs(ns).List(context.TODO(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, remaining.Items, 1)
	assert.Equal(t, "new", remaining.Items[0].Name)
}
