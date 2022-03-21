package pods_test

import (
	"testing"
	"time"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/gc/pods"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// covers MatchesPod phase check
func TestOptions_MatchesPodDoesNotMatchForNonFinishedPhase(t *testing.T) {
	for _, phaseName := range []corev1.PodPhase{corev1.PodRunning, corev1.PodUnknown, corev1.PodPending} {
		testPod := corev1.Pod{Status: corev1.PodStatus{Phase: phaseName}}

		o := pods.Options{}
		result, _ := o.MatchesPod(&testPod)
		assert.False(t, result)
	}
}

// covers MatchesPod date & time checks
func TestOptions_MatchesPodDoesNotMatchForTooYoungPods(t *testing.T) {
	type testVariant struct {
		name                string
		containerFinishedAt time.Time
		maxAge              time.Duration
		shouldMatchPod      bool
	}

	matrix := []testVariant{
		{
			name:                "pod finished yesterday, should be deleted 1h after finish, so true",
			containerFinishedAt: time.Now().Add(time.Hour * 24 * -1), // yesterday
			maxAge:              time.Hour * 1,                       // max 1 hour
			shouldMatchPod:      true,                                // yes, finished yesterday, so older than 1 hour
		},
		{
			name:                "pod finished just now, have remaining 1h to be deleted",
			containerFinishedAt: time.Now(),    // JUST NOW
			maxAge:              time.Hour * 1, // max 1 hour
			shouldMatchPod:      false,         // nope, just finished, will keep for 1 hour still
		},
	}

	for _, variant := range matrix {
		testPod := corev1.Pod{Status: corev1.PodStatus{
			Phase: corev1.PodFailed,
			ContainerStatuses: []corev1.ContainerStatus{
				{State: corev1.ContainerState{
					Terminated: &corev1.ContainerStateTerminated{FinishedAt: metav1.NewTime(variant.containerFinishedAt)},
				}},
			},
		}}

		o := pods.Options{Age: variant.maxAge}
		result, _ := o.MatchesPod(&testPod)
		assert.Equal(t, variant.shouldMatchPod, result, variant.name)
	}
}

func TestCleansUpPodsInProperlyLabelledNamespaces(t *testing.T) {
	expiredStatuses := []corev1.ContainerStatus{
		{State: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{FinishedAt: metav1.NewTime(time.Now().Add(time.Hour * 24 * -1))},
		}},
	}

	client := fake.NewSimpleClientset(
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "other-ns"}},
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name: "jx-test2",
			Labels: map[string]string{
				"gitops.jenkins-x.io/pipeline": "namespaced",
				"env":                          "dev",
			},
		}},

		// and pods belonging to jx-test2, other-ns
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "not-to-be-touched", Namespace: "other-ns"},
			Status:     corev1.PodStatus{Phase: corev1.PodFailed, ContainerStatuses: expiredStatuses},
		},
		&corev1.Pod{ // ONLY this pod should be deleted, as it belongs to namespace that has proper labels
			ObjectMeta: metav1.ObjectMeta{Name: "to-clean-up", Namespace: "jx-test2"},
			Status:     corev1.PodStatus{Phase: corev1.PodFailed, ContainerStatuses: expiredStatuses},
		},
	)

	o := &pods.Options{}
	o.KubeClient = client
	deletedPods, err := o.Run()

	assert.Nil(t, err)
	assert.Equal(t, []string{"jx-test2/to-clean-up"}, deletedPods)
}
