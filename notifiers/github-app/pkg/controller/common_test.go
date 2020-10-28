// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"fmt"
	"io/ioutil"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/yaml"
)

func taskrun() *v1beta1.TaskRun {
	b, err := ioutil.ReadFile("testdata/taskrun.yaml")
	if err != nil {
		panic(fmt.Errorf("error reading input taskrun: %v", err))
	}

	tr := new(v1beta1.TaskRun)
	if err := yaml.Unmarshal(b, tr); err != nil {
		panic(fmt.Errorf("error unmarshalling taskrun: %v", err))
	}
	return tr
}
