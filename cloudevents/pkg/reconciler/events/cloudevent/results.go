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
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type resultMapping struct {
	defaultResultName       string
	annotationResultNameKey string
}

func resultName(objectMeta metav1.Object, mapping resultMapping) string {
	if name, ok := objectMeta.GetAnnotations()[mapping.annotationResultNameKey]; ok {
		return name
	}
	return mapping.defaultResultName
}

func resultFromObjectWithCondition(runObject objectWithCondition, resultName string) (string, error) {
	switch run := runObject.(type) {
	case *v1beta1.TaskRun:
		for _, result := range run.Status.TaskRunResults {
			if result.Name == resultName {
				return result.Value, nil
			}
		}
		return "", fmt.Errorf("no result with name %s found", resultName)
	case *v1beta1.PipelineRun:
		for _, result := range run.Status.PipelineResults {
			if result.Name == resultName {
				return result.Value, nil
			}
		}
		return "", fmt.Errorf("no result with name %s found", resultName)
	}
	return "", fmt.Errorf("unknown type of Tekton resource")
}

func resultForMapping(runObject objectWithCondition, mapping resultMapping) (string, error) {
	name := resultName(runObject.GetObjectMeta(), mapping)
	return resultFromObjectWithCondition(runObject, name)
}
