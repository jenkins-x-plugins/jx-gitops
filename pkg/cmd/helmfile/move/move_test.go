package move_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/move"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
)

func TestUpdateNamespaceInYamlFiles(t *testing.T) {
	tests := []struct {
		folder                  string
		hasReleaseName          bool
		expectedFiles           []string
		expectedHelmAnnotations map[string]string
	}{
		{
			folder:         "output",
			hasReleaseName: false,
			expectedFiles: []string{
				"customresourcedefinitions/jx/lighthouse/lighthousejobs.lighthouse.jenkins.io-crd.yaml",
				"cluster/resources/nginx/nginx-ingress/nginx-ingress-clusterrole.yaml",
				"namespaces/jx/lighthouse/lighthouse-foghorn-deploy.yaml",
			},
		},
		{
			folder:         "dirIncludesReleaseName",
			hasReleaseName: true,
			expectedFiles: []string{
				"customresourcedefinitions/jx/lighthouse/lighthousejobs.lighthouse.jenkins.io-crd.yaml",
				"cluster/resources/nginx/nginx-ingress/nginx-ingress-clusterrole.yaml",
				"namespaces/jx/lighthouse/lighthouse-foghorn-deploy.yaml",
				"customresourcedefinitions/jx/lighthouse-2/lighthousejobs.lighthouse.jenkins.io-crd.yaml",
				"cluster/resources/nginx/nginx-ingress-2/nginx-ingress-clusterrole.yaml",
				"namespaces/jx/lighthouse-2/lighthouse-foghorn-deploy.yaml",
				"namespaces/jx/chart-release/example.yaml",
			},

			expectedHelmAnnotations: map[string]string{
				"namespaces/jx/lighthouse-2/lighthouse-foghorn-deploy.yaml":            "lighthouse-2",
				"cluster/resources/nginx/nginx-ingress/nginx-ingress-clusterrole.yaml": "my-release-name",
			},
		},
	}

	for _, test := range tests {
		tmpDir := t.TempDir()
		_, o := move.NewCmdHelmfileMove()
		t.Logf("generating output to %s\n", tmpDir)
		kubeClient := fake.NewSimpleClientset()
		fakeDiscovery, ok := kubeClient.Discovery().(*fakediscovery.FakeDiscovery)
		if !ok {
			t.Fatalf("couldn't convert Discovery() to *FakeDiscovery")
		}
		fakeDiscovery.Resources = []*v1.APIResourceList{
			{
				TypeMeta:     v1.TypeMeta{},
				GroupVersion: "apps/v1",
				APIResources: []v1.APIResource{
					{
						Name:       "test-deployment",
						Namespaced: true,
						Kind:       "Deployment",
					},
				},
			},
			{
				TypeMeta:     v1.TypeMeta{},
				GroupVersion: "rbac.authorization.k8s.io/v1",
				APIResources: []v1.APIResource{
					{
						Name:       "test-clusterRole",
						Namespaced: false,
						Kind:       "ClusterRole",
					},
				},
			},
			{
				TypeMeta:     v1.TypeMeta{},
				GroupVersion: "example.io/v1",
				APIResources: []v1.APIResource{
					{
						Name:       "test-example",
						Namespaced: true,
						Kind:       "Example",
					},
				},
			},
		}
		o.Dir = filepath.Join("test_data", test.folder)
		o.OutputDir = tmpDir
		o.DirIncludesReleaseName = test.hasReleaseName
		o.AnnotateReleaseNames = true
		o.KubeClient = kubeClient

		err := o.Run()
		require.NoError(t, err, "failed to run helmfile move")

		for _, efn := range test.expectedFiles {
			ef := filepath.Join(append([]string{tmpDir}, strings.Split(efn, "/")...)...)
			assert.FileExists(t, ef)
			t.Logf("generated expected file %s\n", ef)

			if test.expectedHelmAnnotations != nil {
				expectedAnnotation := test.expectedHelmAnnotations[efn]
				if expectedAnnotation != "" {
					u := &unstructured.Unstructured{}
					err = yamls.LoadFile(ef, u)
					require.NoError(t, err, "failed to load %s", ef)
					ann := u.GetAnnotations()
					require.NotNil(t, ann, "should have annotations for file %s", ef)
					annotation := move.HelmReleaseNameAnnotation
					value := ann[annotation]
					assert.Equal(t, expectedAnnotation, value, "for annotation %s in file %s", annotation, ef)
					t.Logf("expected helm annotation is %s\n", value)
				}
			}
		}
	}
}
