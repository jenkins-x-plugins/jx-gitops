package copy_test

import (
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/copy"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCmdCopy(t *testing.T) {
	testCases := []struct {
		name          string
		selector      string
		expectedCount int
	}{
		{
			name:          "thingy",
			expectedCount: 1,
		},
		{
			selector:      "drink=wine",
			expectedCount: 1,
		},
		{
			selector:      "drink=coke",
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		name := "thingy"
		ns := "jx"
		toNS := "cheese"
		key := "drink"
		value := "beer"
		scheme := runtime.NewScheme()
		corev1.AddToScheme(scheme)

		_, o := copy.NewCmdCopy()
		o.Namespace = ns
		o.ToNamespace = toNS
		o.Name = tc.name
		o.Selector = tc.selector
		o.KubeClient = fake.NewSimpleClientset()

		cm := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels: map[string]string{
					"drink": "wine",
				},
			},
			Data: map[string]string{
				key: value,
			},
		}
		o.DynamicClient = dynfake.NewSimpleDynamicClient(scheme, cm)

		err := o.Run()
		require.NoError(t, err, "failed to run the command for query %s", o.Query)

		require.Equal(t, tc.expectedCount, o.Count, "copied ConfigMaps for query %s", o.Query)

		t.Logf("query %s has copied %d resources\n", o.Query, o.Count)
	}
}
