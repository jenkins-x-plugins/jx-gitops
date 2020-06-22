package testhelpers

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx/v2/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertYamlEqual validates YAML without worrying about ordering of keys
func AssertYamlEqual(t *testing.T, expected string, actual string, message string, args ...interface{}) {
	expectedMap := map[string]interface{}{}
	actualMap := map[string]interface{}{}

	reason := fmt.Sprintf(message, args...)

	err := yaml.Unmarshal([]byte(expected), &expectedMap)
	require.NoError(t, err, "failed to unmarshal expected yaml: %s for %s", expected, reason)

	err = yaml.Unmarshal([]byte(actual), &actualMap)
	require.NoError(t, err, "failed to unmarshal actual yaml: %s for %s", actual, reason)

	assert.Equal(t, expectedMap, actualMap, "parsed YAML contents not equal for %s", reason)
}

// AssertTextFilesEqual asserts that the expected file matches the actual file contents
func AssertTextFilesEqual(t *testing.T, expected string, actual string, message string) {
	require.FileExists(t, expected, "expected file for %s", message)
	require.FileExists(t, actual, "actual file for %s", message)

	wantData, err := ioutil.ReadFile(expected)
	require.NoError(t, err, "could not load expected file %s for %s", expected, message)

	gotData, err := ioutil.ReadFile(actual)
	require.NoError(t, err, "could not load actual file %s for %s", actual, message)
	assert.NoError(t, err)

	want := string(wantData)
	got := string(gotData)
	if diff := cmp.Diff(strings.TrimSpace(got), strings.TrimSpace(want)); diff != "" {
		t.Errorf("Unexpected file contents %s for %s", actual, message)
		t.Log(diff)

		t.Logf("generated %s for %s:\n", actual, message)
		t.Logf("\n%s\n", got)
		t.Logf("expected %s for %s:\n", expected, message)
		t.Logf("\n%s\n", want)
	}
}

// AssertLabel asserts the object has the given label value
func AssertLabel(t *testing.T, label string, expected string, objectMeta metav1.ObjectMeta, kindMessage string) {
	message := ObjectNameMessage(objectMeta, kindMessage)
	labels := objectMeta.Labels
	require.NotNil(t, labels, "no labels for %s", message)
	value := labels[label]
	assert.Equal(t, expected, value, "label %s for %s", label, message)
	t.Logf("%s has label %s=%s", message, label, value)
}

// AssertAnnotation asserts the object has the given annotation value
func AssertAnnotation(t *testing.T, annotation string, expected string, objectMeta metav1.ObjectMeta, kindMessage string) {
	message := ObjectNameMessage(objectMeta, kindMessage)
	ann := objectMeta.Annotations
	require.NotNil(t, ann, "no annotations for %s", message)
	value := ann[annotation]
	assert.Equal(t, expected, value, "annotation %s for %s", annotation, message)
	t.Logf("%s has annotation %s=%s", message, annotation, value)
}

// ObjectNameMessage returns an object name message used in the tests
func ObjectNameMessage(objectMeta metav1.ObjectMeta, kindMessage string) string {
	return fmt.Sprintf("%s for name %s", kindMessage, objectMeta.Name)
}

// AssertLabel asserts the object has the given label value
func AssertSecretData(t *testing.T, key string, expected string, secret *corev1.Secret, kindMessage string) {
	require.NotNil(t, secret, "Secret is nil for %s", kindMessage)
	name := secret.Name
	require.NotEmpty(t, secret.Data, "Data is empty in Secret %s for %s", name, kindMessage)

	value := secret.Data[key]
	require.NotNil(t, value, "Secret %s does not have key %s for %s", name, key, kindMessage)
	assert.Equal(t, expected, string(value), "Secret %s key %s for %s", name, key, kindMessage)
	t.Logf("Secret %s has key %s=%s for %s", name, key, value, kindMessage)
}

// AssertFileNotExists asserts that a file does not exist
func AssertFileNotExists(t *testing.T, path string) {
	exists, err := util.FileExists(path)
	require.NoError(t, err, "failed to check if file exists %s", path)
	assert.False(t, exists, "file should not exist %s", path)
}
