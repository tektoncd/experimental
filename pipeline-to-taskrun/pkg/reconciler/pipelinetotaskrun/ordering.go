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

package pipelinetotaskrun

import (
	"fmt"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

type linkedTask struct {
	task *v1beta1.PipelineTask
	next *linkedTask
}

func putTasksInOrder(tasks []v1beta1.PipelineTask) ([]v1beta1.PipelineTask, error) {
	seen := map[string]*linkedTask{}
	var root *linkedTask

	for i := 0; i < len(tasks); i++ {
		seen[tasks[i].Name] = &linkedTask{task: &tasks[i]}
	}

	for _, task := range tasks {
		if len(task.RunAfter) > 1 {
			return nil, fmt.Errorf("fan in not yet supported but %s has more than one runAfter", task.Name)
		}
		l, _ := seen[task.Name]
		if len(task.RunAfter) == 0 {
			if root != nil {
				return nil, fmt.Errorf("parallel tasks not yet supported by %s and %s are trying to run in parallel", task.Name, root.task.Name)
			} else {
				root = l
			}
		} else {
			before, ok := seen[task.RunAfter[0]]
			if !ok {
				return nil, fmt.Errorf("task %s trying to run after task %s which is not present", task.Name, task.RunAfter[0])
			}
			before.next = l
		}
	}
	if root == nil {
		return nil, fmt.Errorf("invalid sequence, there was no starting task (probably a loop?)")
	}

	ordered := []v1beta1.PipelineTask{*root.task}
	curr := root.next
	for {
		if curr == nil {
			if len(ordered) < len(tasks) {
				return nil, fmt.Errorf("sequence was not completely connected, gap after %s", ordered[len(ordered)-1].Name)
			}
			break
		}
		ordered = append(ordered, *curr.task)
		curr = curr.next
	}

	return ordered, nil
}
