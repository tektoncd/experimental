/*
Copyright 2019 The Tekton Authors

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

package config_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	test "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test/diff"
)

func TestEquals(t *testing.T) {
	testCases := []struct {
		name     string
		left     *config.Defaults
		right    *config.Defaults
		expected bool
	}{
		{
			name:     "left and right nil",
			left:     nil,
			right:    nil,
			expected: true,
		},
		{
			name:     "left nil",
			left:     nil,
			right:    &config.Defaults{},
			expected: false,
		},
		{
			name:     "right nil",
			left:     &config.Defaults{},
			right:    nil,
			expected: false,
		},
		{
			name:     "right and right default",
			left:     &config.Defaults{},
			right:    &config.Defaults{},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.left.Equals(tc.right)
			if actual != tc.expected {
				t.Errorf("Comparison failed expected: %t, actual: %t", tc.expected, actual)
			}
		})
	}
}

func verifyConfigFileWithExpectedConfig(t *testing.T, fileName string, expectedConfig *config.Defaults) {
	t.Helper()
	cm := test.ConfigMapFromTestFile(t, fileName)
	if Defaults, err := config.NewDefaultsFromConfigMap(cm); err == nil {
		if d := cmp.Diff(Defaults, expectedConfig); d != "" {
			t.Errorf("Diff:\n%s", diff.PrintWantGot(d))
		}
	} else {
		t.Errorf("NewDefaultsFromConfigMap(actual) = %v", err)
	}
}

func verifyConfigFileWithExpectedError(t *testing.T, fileName string) {
	cm := test.ConfigMapFromTestFile(t, fileName)
	if _, err := config.NewDefaultsFromConfigMap(cm); err == nil {
		t.Errorf("NewDefaultsFromConfigMap(actual) was expected to return an error")
	}
}
