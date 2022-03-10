package pods

import (
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

// covers MatchesPod phase check
func TestOptions_MatchesPodDoesNotMatchForNonFinishedPhase(t *testing.T) {
	for _, phaseName := range []corev1.PodPhase{corev1.PodRunning, corev1.PodUnknown, corev1.PodPending} {
		testPod := corev1.Pod{Status: corev1.PodStatus{Phase: phaseName}}

		o := Options{}
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

		o := Options{Age: variant.maxAge}
		result, _ := o.MatchesPod(&testPod)
		assert.Equal(t, variant.shouldMatchPod, result, variant)
	}
}
