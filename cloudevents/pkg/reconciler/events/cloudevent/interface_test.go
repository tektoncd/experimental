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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis"
)

// myObjectWithCondition is an objectWithCondition that is not PipelineRUn or TaskRun
type myObjectWithCondition struct{}

func (mowc myObjectWithCondition) DeepCopyObject() runtime.Object             { return nil }
func (mowc myObjectWithCondition) GetObjectKind() schema.ObjectKind           { return nil }
func (mowc myObjectWithCondition) GetObjectMeta() metav1.Object               { return nil }
func (mowc myObjectWithCondition) GetStatusCondition() apis.ConditionAccessor { return nil }
