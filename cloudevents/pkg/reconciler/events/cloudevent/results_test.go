/*
Copyright 2021 The Tekton Authors

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

package cloudevent

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/test/diff"
	corev1 "k8s.io/api/core/v1"
)

func TestResultFromObjectWithCondition(t *testing.T) {
	var myObject myObjectWithCondition
	myObject = myObjectWithCondition{}
	resultTests := []struct {
		desc       string
		object     interface{}
		resultName string
		wantResult string
		wantError  bool
	}{{
		desc:       "not a taskrun or pipelinerun",
		object:     &myObject,
		resultName: "foobar",
		wantResult: "",
		wantError:  true,
	}, {
		desc: "pipelinerun with result",
		object: createPipelineRunWithConditionAndResults(
			corev1.ConditionUnknown,
			"somethingsomething",
			map[string]string{},
			map[string]string{"test1": "value1", "test2": "value2"}),
		resultName: "test2",
		wantResult: "value2",
		wantError:  false,
	}, {
		desc: "pipelinerun without result",
		object: createPipelineRunWithConditionAndResults(
			corev1.ConditionUnknown,
			"somethingsomething",
			map[string]string{},
			map[string]string{"test1": "value1", "test2": "value2"}),
		resultName: "missing",
		wantResult: "",
		wantError:  true,
	}, {
		desc: "taskrun with result",
		object: createTaskRunWithConditionAndResults(
			corev1.ConditionUnknown,
			"somethingsomething",
			map[string]string{},
			map[string]string{"test1": "value1", "test2": "value2"}),
		resultName: "test2",
		wantResult: "value2",
		wantError:  false,
	}, {
		desc: "taskrun without result",
		object: createTaskRunWithConditionAndResults(
			corev1.ConditionUnknown,
			"somethingsomething",
			map[string]string{},
			map[string]string{"test1": "value1", "test2": "value2"}),
		resultName: "missing",
		wantResult: "",
		wantError:  true,
	}}

	for _, c := range resultTests {
		t.Run(c.desc, func(t *testing.T) {

			got, err := resultFromObjectWithCondition(c.object.(objectWithCondition), c.resultName)
			if err != nil {
				if !c.wantError {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				if c.wantError {
					t.Fatalf("I did expect an error but I got %s", got)
				}
				if d := cmp.Diff(c.wantResult, got); d != "" {
					t.Errorf("Wrong result %s", diff.PrintWantGot(d))
				}
			}
		})
	}
}
