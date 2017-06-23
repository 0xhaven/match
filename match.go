// Package match provides a set of functions for deep comparison of two values.
package match

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"reflect"
)

// MatchesHTTPResponse determines whether two http.Responses match.
func MatchesHTTPResponse(expected, actual *http.Response) bool {
	if expected.StatusCode != actual.StatusCode {
		return false
	}

	if !Matches(expected.Header, actual.Header) {
		return false
	}

	expBody, expErr := ioutil.ReadAll(expected.Body)
	actBody, actErr := ioutil.ReadAll(actual.Body)
	if expErr != nil || actErr != nil {
		return expErr == actErr
	}

	if !Matches(expected.Trailer, actual.Trailer) {
		return false
	}

	// Attempt to JSON unmarshal.
	var exp, act interface{}
	if err := json.Unmarshal(expBody, &exp); err != nil {
		return string(expBody) == string(actBody)
	}
	if err := json.Unmarshal(actBody, &act); err != nil {
		return false
	}

	return Matches(exp, act)
}

// Matches determines whether two arbitrary interfaces match.
func Matches(expected, actual interface{}) bool {
	return matchesValues(reflect.ValueOf(expected), reflect.ValueOf(actual))
}

// matchesValues recursively requires deep equality (order-agnositic equality for slices).
func matchesValues(expected, actual reflect.Value) bool {
	if expected.Type() != actual.Type() {
		return false
	}
	switch expected.Kind() {
	case reflect.Ptr:
		if expected.IsNil() || actual.IsNil() {
			return expected.IsNil() && actual.IsNil()
		}
		return matchesValues(expected.Elem(), actual.Elem())
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		if expected.Len() != actual.Len() {
			return false
		}
		used := make(map[int]bool)
		for i := 0; i < expected.Len(); i++ {
			found := false
			for j := 0; j < actual.Len(); j++ {
				if used[j] {
					continue
				}
				if matchesValues(expected.Index(i), actual.Index(j)) {
					used[j] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	case reflect.Struct:
		for i := 0; i < expected.NumField(); i++ {
			if expected.Type().Field(i).PkgPath != "" {
				// skip unexported fields
				continue
			}
			if !matchesValues(expected.Field(i), actual.Field(i)) {
				return false
			}
		}
	case reflect.Map:
		for _, key := range expected.MapKeys() {
			if !matchesValues(expected.MapIndex(key), actual.MapIndex(key)) {
				return false
			}
		}
	default:
		return expected.Interface() == actual.Interface()
	}
	return true
}
