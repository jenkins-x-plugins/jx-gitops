/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mapslices_test

import (
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/mapslices"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestGetString(t *testing.T) {
	obj := yaml.MapSlice{
		yaml.MapItem{
			Key: "a",
			Value: yaml.MapSlice{
				{
					Key:   "b",
					Value: "cheese",
				},
				{
					Key:   "c",
					Value: nil,
				},
				{
					Key:   "d",
					Value: []interface{}{"foo"},
				},
			},
		},
		yaml.MapItem{
			Key:   "topLevel",
			Value: "thingy",
		},
	}

	testCases := []struct {
		fields   []string
		expected interface{}
	}{
		{
			fields:   []string{"topLevel"},
			expected: "thingy",
		},
		{
			fields:   []string{"a", "b"},
			expected: "cheese",
		},
	}

	for _, tc := range testCases {
		value, _, err := mapslices.NestedString(obj, tc.fields...)
		require.NoError(t, err, "failed to find nested strings %#v", tc.fields)
		assert.Equal(t, tc.expected, value, "for nested strings %#v", tc.fields)
	}
}

func TestSetNestedStrings(t *testing.T) {
	obj := yaml.MapSlice{
		yaml.MapItem{
			Key: "a",
			Value: yaml.MapSlice{
				{
					Key:   "b",
					Value: "cheese",
				},
				{
					Key:   "c",
					Value: nil,
				},
				{
					Key:   "d",
					Value: []interface{}{"foo"},
				},
			},
		},
		yaml.MapItem{
			Key: "b",
			Value: yaml.MapSlice{
				{
					Key:   "b",
					Value: "cheese",
				},
			},
		},
		yaml.MapItem{
			Key:   "topLevel",
			Value: "thingy",
		},
	}

	testCases := []struct {
		fields []string
		value  interface{}
		flag   bool
		check  func(t *testing.T, fields []string)
	}{
		{
			fields: []string{"a", "a"},
			value:  "first-child",
			flag:   true,
			check: func(t *testing.T, fields []string) {
				AssertMapSliceIndex(t, obj, 0, fields...)
			},
		},
		{
			fields: []string{"b", "a"},
			value:  "first-child",
			flag:   true,
			check: func(t *testing.T, fields []string) {
				AssertMapSliceIndex(t, obj, 0, fields...)
			},
		},
		{
			fields: []string{"a", "b2"},
			value:  "third-child",
			flag:   true,
			check: func(t *testing.T, fields []string) {
				AssertMapSliceIndex(t, obj, 2, fields...)
			},
		},
		{
			fields: []string{"a", "b"},
			value:  "cheese",
			flag:   false,
		},
		{
			fields: []string{"a", "newkey"},
			value:  "newkey-value",
			flag:   true,
		},
		{
			fields: []string{"topLevel"},
			value:  "newthing",
			flag:   true,
		},
		{
			fields: []string{"a", "b"},
			value:  "updated-cheese",
			flag:   true,
		},
		{
			fields: []string{"a", "zzz"},
			value:  "last-child",
			flag:   true,
			check: func(t *testing.T, fields []string) {
				AssertMapSliceIndex(t, obj, 6, fields...)
			},
		},
	}

	var err error
	flag := false
	for _, tc := range testCases {
		obj, flag, err = mapslices.SetNestedField(obj, tc.value, tc.fields...)
		require.NoError(t, err, "failed to set nested strings %#v to %#v", tc.fields, tc.value)

		data, err := yaml.Marshal(obj)
		assert.NoError(t, err, "failed to marshal as YAML for nested strings %#v", tc.fields)
		t.Logf("set flags %#v = %#v modified %v to make YAML:\n%s\n", tc.fields, tc.value, tc.flag, string(data))

		value, _, err := mapslices.NestedString(obj, tc.fields...)
		require.NoError(t, err, "failed to find nested strings %#v", tc.fields)

		assert.Equal(t, tc.value, value, "for nested strings %#v", tc.fields)
		assert.Equal(t, tc.flag, flag, "modified flag for nested strings %#v", tc.fields)

		if tc.check != nil {
			tc.check(t, tc.fields)
		}
	}
}

func TestRemoveNestedField(t *testing.T) {
	obj := yaml.MapSlice{
		yaml.MapItem{
			Key: "x",
			Value: yaml.MapSlice{
				{
					Key:   "y",
					Value: 1,
				},
				{
					Key:   "a",
					Value: "foo",
				},
			},
		},
	}
	var flag bool
	obj, flag = mapslices.RemoveNestedField(obj, "x", "a")
	AssertLen(t, obj, "x", 1)
	assert.True(t, flag, "should have removed field")

	obj, flag = mapslices.RemoveNestedField(obj, "x", "y")
	AssertLen(t, obj, "x", 0)
	AssertEmpty(t, obj, "x")
	assert.True(t, flag, "should have removed field")

	obj, flag = mapslices.RemoveNestedField(obj, "x")
	assert.Empty(t, obj)
	assert.True(t, flag, "should have removed field")

	obj, flag = mapslices.RemoveNestedField(obj, "x") // Remove of a non-existent field
	assert.Empty(t, obj)
	assert.False(t, flag, "should have removed field")
}

func TestNestedField(t *testing.T) {
	target := map[string]interface{}{"foo": "bar"}

	obj := yaml.MapSlice{
		yaml.MapItem{
			Key: "a",
			Value: yaml.MapSlice{
				{
					Key:   "b",
					Value: target,
				},
				{
					Key:   "c",
					Value: nil,
				},
				{
					Key:   "d",
					Value: []interface{}{"foo"},
				},
			},
		},
	}

	// case 1: field exists and is non-nil
	res, exists, err := mapslices.NestedField(obj, "a", "b")
	assert.True(t, exists)
	assert.Nil(t, err)
	assert.Equal(t, target, res)
	target["foo"] = "baz"
	assert.Equal(t, target["foo"], res.(map[string]interface{})["foo"], "result should be a reference to the expected item")

	// case 2: field exists and is nil
	res, exists, err = mapslices.NestedField(obj, "a", "c")
	assert.True(t, exists)
	assert.Nil(t, err)
	assert.Nil(t, res)

	// case 3: error traversing obj
	res, exists, err = mapslices.NestedField(obj, "a", "d", "foo")
	assert.False(t, exists)
	assert.NotNil(t, err)
	assert.Nil(t, res)

	// case 4: field does not exist
	res, exists, err = mapslices.NestedField(obj, "a", "e")
	assert.False(t, exists)
	assert.Nil(t, err)
	assert.Nil(t, res)
}

// AssertMapSliceIndex asserts that the field is at the specified index in its parent
func AssertMapSliceIndex(t *testing.T, obj yaml.MapSlice, expectedIndex int, fields ...string) {
	i := len(fields) - 1
	require.True(t, i >= 0, "no fields specified")

	if i > 0 {
		parent, flag, err := mapslices.NestedMapSlice(obj, fields[0:i]...)
		require.NoError(t, err, "could not find MapSlice at %#v", fields[0:i])
		require.True(t, flag, "no MapSlice at %#v", fields[0:i])
		require.NotNil(t, parent, "nil MapSlice at %#v", fields[0:i])
		obj = parent
	}

	field := fields[i]
	idx := mapslices.IndexOf(obj, field)
	assert.Equal(t, expectedIndex, idx, "expected index for field %s in object %#v", field, obj)
}

func AssertEmpty(t *testing.T, obj yaml.MapSlice, field interface{}) {
	val := AssertLen(t, obj, field, 0)
	assert.Empty(t, val, "field %#v has value %#v for %#v", field, val, obj)

}

func AssertLen(t *testing.T, obj yaml.MapSlice, field interface{}, expected int) interface{} {
	val, flag := mapslices.Get(obj, field)
	assert.True(t, flag, "did not find field %#v for object %#v", field, obj)
	assert.Len(t, val, expected, "field %#v has value %#v for %#v", field, val, obj)
	return val
}
