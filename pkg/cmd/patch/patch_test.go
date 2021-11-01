package patch_test

import (
	"context"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/patch"
	"github.com/jenkins-x/jx-helpers/v3/pkg/knative_pkg/duck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynfake "k8s.io/client-go/dynamic/fake"
)

func TestPatch(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)

	ns := "jx"
	_, o := patch.NewCmdPatch()
	o.Namespace = ns
	o.Selector = "drink=wine"
	o.Data = `{"spec":{"template":{"metadata":{"annotations":{"cheese": "edam"}}}}}`

	name := "cheese"
	d1 := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
			Labels: map[string]string{
				"drink": "wine",
			},
		},
		Spec: v1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"something": "whatnot"},
				},
			},
		},
	}
	d2 := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "another",
			Namespace: ns,
			Labels: map[string]string{
				"drink": "beer",
			},
		},
		Spec: v1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"something": "whatnot"},
				},
			},
		},
	}
	o.DynamicClient = dynfake.NewSimpleDynamicClient(scheme, d1, d2)

	err := o.Run()
	require.NoError(t, err, "failed to run the command")

	ctx := context.TODO()
	r := GetDeployment(ctx, t, o, d1.Name)
	assert.Equal(t, "edam", r.Spec.Template.ObjectMeta.Annotations["cheese"], "d1 should have the cheese annotation")

	r = GetDeployment(ctx, t, o, d2.Name)
	assert.Equal(t, "", r.Spec.Template.ObjectMeta.Annotations["cheese"], "d2 should not have the cheese annotation")
}

func GetDeployment(ctx context.Context, t *testing.T, o *patch.Options, resourceName string) *v1.Deployment {
	versionResource := o.GetGroupVersion()
	r, err := o.DynamicClient.Resource(versionResource).Namespace(o.Namespace).Get(ctx, resourceName, metav1.GetOptions{})
	require.NoError(t, err, "failed to find resource %s in namespace %s", resourceName, o.Namespace)

	r2 := &v1.Deployment{}
	err = duck.FromUnstructured(r, r2)
	require.NoError(t, err, "failed to convert resource %s to Deployment. Got %#v", resourceName, r)
	return r2
}
