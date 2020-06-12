/*
Copyright 2015 The Kubernetes Authors.

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

package mapslices

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

// IndexOf returns the index of the item with the given key or -1 if its not found
func IndexOf(obj yaml.MapSlice, key interface{}) int {
	for i, s := range obj {
		if cmp.Equal(s.Key, key) {
			return i
		}
	}
	return -1
}

// Get returns the value of the slice with the given key
func Get(obj yaml.MapSlice, key interface{}) (interface{}, bool) {
	for _, s := range obj {
		if cmp.Equal(s.Key, key) {
			return s.Value, true
		}
	}
	return nil, false
}

// Set sets the key to the given value. Returns true if a modification was made
func Set(obj yaml.MapSlice, key interface{}, value interface{}) (yaml.MapSlice, bool) {
	index := 0
	doCompare := true
	for i, s := range obj {
		if cmp.Equal(s.Key, key) {
			if cmp.Equal(s.Value, value) {
				return obj, false
			}
			obj[i].Value = value
			return obj, true
		}
		if doCompare {
			if cmpLess(s.Key, key) {
				index = i + 1
			} else {
				doCompare = false
			}
		}
	}

	item := yaml.MapItem{
		Key:   key,
		Value: value,
	}
	if index >= len(obj) {
		obj = append(obj, item)
	} else {
		// lets insert at the given index
		obj = append(obj[:index+1], obj[index:]...)
		obj[index] = item
	}
	return obj, true
}

// cmpLess compares strings and numbers in order
func cmpLess(v1 interface{}, v2 interface{}) bool {
	t1, ok := v1.(string)
	if ok {
		t2, ok := v2.(string)
		if ok {
			return t1 < t2
		}
	}
	i1, ok := v1.(int)
	if ok {
		i2, ok := v2.(int)
		if ok {
			return i1 < i2
		}
	}
	return false
}

// Deletes the key in the slice. Returns true if a modification was made
func Delete(obj yaml.MapSlice, key interface{}) (yaml.MapSlice, bool) {
	for i, s := range obj {
		if cmp.Equal(s.Key, key) {
			obj = append(obj[0:i], obj[i+1:]...)
			return obj, true
		}
	}
	return obj, false
}

// NestedField returns a reference to a nested field.
// Returns false if value is not found and an error if unable
// to traverse obj.
func NestedField(obj yaml.MapSlice, fields ...string) (interface{}, bool, error) {
	var val interface{} = obj
	flag := false
	for i, field := range fields {
		if m, ok := val.(yaml.MapSlice); ok {
			val, flag = Get(m, field)
			if val == nil {
				return nil, flag, nil
			}
		} else {
			return nil, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected yaml.MapSlice", jsonPath(fields[:i+1]), val, val)
		}
	}
	return val, true, nil
}

// NestedString returns the string value of a nested field.
// Returns false if value is not found and an error if not a string.
func NestedString(obj yaml.MapSlice, fields ...string) (string, bool, error) {
	val, found, err := NestedField(obj, fields...)
	if !found || err != nil {
		return "", found, err
	}
	s, ok := val.(string)
	if !ok {
		return "", false, fmt.Errorf("%v accessor error: %v is of the type %T, expected string", jsonPath(fields), val, val)
	}
	return s, true, nil
}

// NestedBool returns the bool value of a nested field.
// Returns false if value is not found and an error if not a bool.
func NestedBool(obj yaml.MapSlice, fields ...string) (bool, bool, error) {
	val, found, err := NestedField(obj, fields...)
	if !found || err != nil {
		return false, found, err
	}
	b, ok := val.(bool)
	if !ok {
		return false, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected bool", jsonPath(fields), val, val)
	}
	return b, true, nil
}

// NestedFloat64 returns the float64 value of a nested field.
// Returns false if value is not found and an error if not a float64.
func NestedFloat64(obj yaml.MapSlice, fields ...string) (float64, bool, error) {
	val, found, err := NestedField(obj, fields...)
	if !found || err != nil {
		return 0, found, err
	}
	f, ok := val.(float64)
	if !ok {
		return 0, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected float64", jsonPath(fields), val, val)
	}
	return f, true, nil
}

// NestedInt64 returns the int64 value of a nested field.
// Returns false if value is not found and an error if not an int64.
func NestedInt64(obj yaml.MapSlice, fields ...string) (int64, bool, error) {
	val, found, err := NestedField(obj, fields...)
	if !found || err != nil {
		return 0, found, err
	}
	i, ok := val.(int64)
	if !ok {
		return 0, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected int64", jsonPath(fields), val, val)
	}
	return i, true, nil
}

// NestedStringSlice returns a copy of []string value of a nested field.
// Returns false if value is not found and an error if not a []interface{} or contains non-string items in the slice.
func NestedStringSlice(obj yaml.MapSlice, fields ...string) ([]string, bool, error) {
	val, found, err := NestedField(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	m, ok := val.([]interface{})
	if !ok {
		return nil, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected []interface{}", jsonPath(fields), val, val)
	}
	strSlice := make([]string, 0, len(m))
	for _, v := range m {
		if str, ok := v.(string); ok {
			strSlice = append(strSlice, str)
		} else {
			return nil, false, fmt.Errorf("%v accessor error: contains non-string key in the slice: %v is of the type %T, expected string", jsonPath(fields), v, v)
		}
	}
	return strSlice, true, nil
}

/*
// NestedStringMap returns a copy of map[string]string value of a nested field.
// Returns false if value is not found and an error if not a yaml.MapSlice or contains non-string values in the map.
func NestedStringMap(obj yaml.MapSlice, fields ...string) (map[string]string, bool, error) {
	m, found, err := NestedMapSlice(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	strMap := make(map[string]string, len(m))
	for k, v := range m {
		if str, ok := v.(string); ok {
			strMap[k] = str
		} else {
			return nil, false, fmt.Errorf("%v accessor error: contains non-string key in the map: %v is of the type %T, expected string", jsonPath(fields), v, v)
		}
	}
	return strMap, true, nil
}
*/

// NestedMapSlice returns a yaml.MapSlice value of a nested field.
// Returns false if value is not found and an error if not a yaml.MapSlice.
func NestedMapSlice(obj yaml.MapSlice, fields ...string) (yaml.MapSlice, bool, error) {
	val, found, err := NestedField(obj, fields...)
	if !found || err != nil {
		return nil, found, err
	}
	m, ok := val.(yaml.MapSlice)
	if !ok {
		return nil, false, fmt.Errorf("%v accessor error: %v is of the type %T, expected yaml.MapSlice", jsonPath(fields), val, val)
	}
	return m, true, nil
}

// SetNestedField sets the value of a nested field to a deep copy of the value provided.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedField(obj yaml.MapSlice, value interface{}, fields ...string) (yaml.MapSlice, bool, error) {
	m := obj
	parent := obj
	flag := false
	idx := len(fields) - 1
	field := ""
	i := 0
	for i, field = range fields[:idx] {
		if val, _ := Get(m, field); val != nil {
			if valMap, ok := val.(yaml.MapSlice); ok {
				parent = m
				m = valMap
			} else {
				return obj, false, fmt.Errorf("value cannot be set because %v is not a yaml.MapSlice", jsonPath(fields[:i+1]))
			}
		} else {
			newVal := yaml.MapSlice{}
			Set(m, field, newVal)
			m = newVal
		}
	}
	m, flag = Set(m, fields[idx], value)
	if idx == 0 {
		return m, flag, nil
	}
	Set(parent, field, m)
	return obj, flag, nil
}

// SetNestedStringSlice sets the string slice value of a nested field.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedStringSlice(obj yaml.MapSlice, value []string, fields ...string) (yaml.MapSlice, bool, error) {
	m := make([]interface{}, 0, len(value)) // convert []string into []interface{}
	for _, v := range value {
		m = append(m, v)
	}
	return SetNestedField(obj, m, fields...)
}

// SetNestedSlice sets the slice value of a nested field.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedSlice(obj yaml.MapSlice, value []interface{}, fields ...string) (yaml.MapSlice, bool, error) {
	return SetNestedField(obj, value, fields...)
}

// SetNestedStringMap sets the map[string]string value of a nested field.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedStringMap(obj yaml.MapSlice, value map[string]string, fields ...string) (yaml.MapSlice, bool, error) {
	m := make(yaml.MapSlice, len(value)) // convert map[string]string into yaml.MapSlice
	for k, v := range value {
		Set(m, k, v)
	}
	return SetNestedField(obj, m, fields...)
}

// SetNestedMap sets the yaml.MapSlice value of a nested field.
// Returns an error if value cannot be set because one of the nesting levels is not a yaml.MapSlice.
func SetNestedMap(obj yaml.MapSlice, value yaml.MapSlice, fields ...string) (yaml.MapSlice, bool, error) {
	return SetNestedField(obj, value, fields...)
}

// RemoveNestedField removes the nested field from the obj.
func RemoveNestedField(obj yaml.MapSlice, fields ...string) (yaml.MapSlice, bool) {
	m := obj
	parent := obj
	idx := len(fields) - 1
	field := ""
	for _, field = range fields[:idx] {
		val, _ := Get(m, field)
		if x, ok := val.(yaml.MapSlice); ok {
			parent = m
			m = x
		} else {
			return obj, false
		}
	}
	v, flag := Delete(m, fields[idx])
	if idx == 0 {
		return v, flag
	}
	Set(parent, field, v)
	return obj, flag
}

func jsonPath(fields []string) string {
	return "." + strings.Join(fields, ".")
}
