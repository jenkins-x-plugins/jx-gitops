package move_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/move"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type test struct {
	folder                         string
	hasReleaseName                 bool
	expectedFiles                  []string
	expectedHelmReleaseAnnotations map[string]string
	expectedNamespace              map[string]string
	nonStandardNamespace           map[string]string
	overrideNamespace              bool
}

func TestUpdateNamespaceInYamlFiles(t *testing.T) {
	tests := []test{
		{
			folder:            "output",
			hasReleaseName:    false,
			overrideNamespace: true,

			expectedFiles: []string{
				"customresourcedefinitions/jx/lighthouse/lighthousejobs.lighthouse.jenkins.io-crd.yaml",
				"cluster/resources/nginx/nginx-ingress/nginx-ingress-clusterrole.yaml",
				"namespaces/jx/lighthouse/lighthouse-foghorn-deploy.yaml",
			},
			expectedNamespace: map[string]string{
				"customresourcedefinitions/jx/lighthouse/lighthousejobs.lighthouse.jenkins.io-crd.yaml": "jx",
				"cluster/resources/nginx/nginx-ingress/nginx-ingress-clusterrole.yaml":                  "nginx",
				"namespaces/jx/lighthouse/lighthouse-foghorn-deploy.yaml":                               "jx",
			},
		},
		{
			folder:            "dirIncludesReleaseName",
			hasReleaseName:    true,
			overrideNamespace: true,

			expectedFiles: []string{
				"customresourcedefinitions/jx/lighthouse/lighthousejobs.lighthouse.jenkins.io-crd.yaml",
				"cluster/resources/nginx/nginx-ingress/nginx-ingress-clusterrole.yaml",
				"namespaces/jx/lighthouse/lighthouse-foghorn-deploy.yaml",
				"customresourcedefinitions/jx/lighthouse-2/lighthousejobs.lighthouse.jenkins.io-crd.yaml",
				"cluster/resources/nginx/nginx-ingress-2/nginx-ingress-clusterrole.yaml",
				"namespaces/jx/lighthouse-2/lighthouse-foghorn-deploy.yaml",
				"namespaces/jx/chart-release/example.yaml",
			},
			expectedHelmReleaseAnnotations: map[string]string{
				"namespaces/jx/lighthouse-2/lighthouse-foghorn-deploy.yaml":            "lighthouse-2",
				"cluster/resources/nginx/nginx-ingress/nginx-ingress-clusterrole.yaml": "my-release-name",
			},
			expectedNamespace: map[string]string{
				"namespaces/jx/lighthouse-2/lighthouse-foghorn-deploy.yaml":                               "jx",
				"cluster/resources/nginx/nginx-ingress/nginx-ingress-clusterrole.yaml":                    "nginx",
				"customresourcedefinitions/jx/lighthouse/lighthousejobs.lighthouse.jenkins.io-crd.yaml":   "jx",
				"namespaces/jx/lighthouse/lighthouse-foghorn-deploy.yaml":                                 "jx",
				"customresourcedefinitions/jx/lighthouse-2/lighthousejobs.lighthouse.jenkins.io-crd.yaml": "jx",
				"cluster/resources/nginx/nginx-ingress-2/nginx-ingress-clusterrole.yaml":                  "nginx",
				"namespaces/jx/chart-release/example.yaml":                                                "jx",
			},
		},
		{
			folder:            "nonStandardNamespace",
			hasReleaseName:    true,
			overrideNamespace: true,
			expectedFiles: []string{
				"namespaces/selenium-grid/keda-selenium/keda-operator-auth-reader-rb.yaml",
			},
			expectedHelmReleaseAnnotations: map[string]string{
				"namespaces/selenium-grid/keda-selenium/keda-operator-auth-reader-rb.yaml": "selenium",
			},
			expectedNamespace: map[string]string{
				"namespaces/selenium-grid/keda-selenium/keda-operator-auth-reader-rb.yaml": "selenium-grid",
			},
		},
		{
			folder:            "nonStandardNamespace",
			hasReleaseName:    true,
			overrideNamespace: false,
			expectedFiles: []string{
				"namespaces/kube-system/keda-selenium/keda-operator-auth-reader-rb.yaml",
			},
			expectedHelmReleaseAnnotations: map[string]string{
				"namespaces/kube-system/keda-selenium/keda-operator-auth-reader-rb.yaml": "selenium",
			},
			expectedNamespace: map[string]string{
				"namespaces/kube-system/keda-selenium/keda-operator-auth-reader-rb.yaml": "selenium-grid",
			},
			nonStandardNamespace: map[string]string{
				"namespaces/kube-system/keda-selenium/keda-operator-auth-reader-rb.yaml": "kube-system",
			},
		},
	}

	for _, test := range tests {
		testMove(t, &test)
	}
}

func testMove(t *testing.T, test *test) {
	_, o := move.NewCmdHelmfileMove()

	o.Dir = filepath.Join("testdata", test.folder)
	o.DirIncludesReleaseName = test.hasReleaseName

	tmpDir := t.TempDir()
	t.Logf("generating output to namespace %s, override namespace: %t\n", tmpDir, test.overrideNamespace)
	o.OutputDir = tmpDir
	o.OverrideNamespace = test.overrideNamespace

	err := o.Run()
	require.NoError(t, err, "failed to run helmfile move")

	for _, efn := range test.expectedFiles {
		ef := filepath.Join(append([]string{tmpDir}, strings.Split(efn, "/")...)...)
		assert.FileExists(t, ef)
		t.Logf("generated expected file %s\n", ef)

		if test.expectedHelmReleaseAnnotations != nil {
			expectedAnnotation := test.expectedHelmReleaseAnnotations[efn]
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
		if test.expectedNamespace != nil {
			expectedNS := test.expectedNamespace[efn]
			if expectedNS != "" {
				u := &unstructured.Unstructured{}
				err = yamls.LoadFile(ef, u)
				require.NoError(t, err, "failed to load %s", ef)
				ann := u.GetAnnotations()
				require.NotNil(t, ann, "should have annotations for file %s", ef)
				annotation := move.HelmReleaseNameSpaceAnnotation
				value := ann[annotation]
				assert.Equal(t, expectedNS, value, "for annotation %s in file %s", annotation, ef)
				t.Logf("expected namespace annotation is %s\n", value)
				nonStandardNS, hasNonStandardNS := test.nonStandardNamespace[efn]
				ns := u.GetNamespace()
				if test.overrideNamespace || !hasNonStandardNS {
					if ns != "" {
						assert.Equal(t, expectedNS, ns, "for namespace %s in file %s", annotation, ef)
					}
				} else {
					assert.Equal(t, nonStandardNS, ns, "for namespace %s in file %s", annotation, ef)
				}
			}
		}
	}
}
