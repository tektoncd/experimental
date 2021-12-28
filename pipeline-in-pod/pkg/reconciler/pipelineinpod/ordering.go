/*
This is copied from the experimental pipeline -> taskrun
*/
package pipelineinpod

import (
	"fmt"
)

type linkedTask struct {
	task *pipelineTaskContainers
	next *linkedTask
}

func putTasksInOrder(ptcs []pipelineTaskContainers) ([]pipelineTaskContainers, error) {
	seen := map[string]*linkedTask{}
	var root *linkedTask

	for i := 0; i < len(ptcs); i++ {
		seen[ptcs[i].pt.Name] = &linkedTask{task: &ptcs[i]}
	}

	for _, ptc := range ptcs {
		task := ptc.pt
		if len(task.RunAfter) > 1 {
			return nil, fmt.Errorf("fan in not yet supported but %s has more than one runAfter", task.Name)
		}
		l, _ := seen[task.Name]
		if len(task.RunAfter) == 0 {
			if root != nil {
				return nil, fmt.Errorf("parallel tasks not yet supported by %s and %s are trying to run in parallel", task.Name, root.task.pt.Name)
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

	ordered := []pipelineTaskContainers{*root.task}
	curr := root.next
	for {
		if curr == nil {
			if len(ordered) < len(ptcs) {
				return nil, fmt.Errorf("sequence was not completely connected, gap after %s", ordered[len(ordered)-1].pt.Name)
			}
			break
		}
		ordered = append(ordered, *curr.task)
		curr = curr.next
	}

	return ordered, nil
}

/*
func getOrderedSteps(tasks []v1beta1.PipelineTask) ([]v1beta1.Step, error) {
	orderedTasks, err := putTasksInOrder(tasks)
	if err != nil {
		return nil, err
	}
	orderedSteps := make([]v1beta1.Step, 0)
	for _, pt := range orderedTasks {
		steps, err := v1beta1.MergeStepsWithStepTemplate(pt.TaskSpec.StepTemplate, pt.TaskSpec.Steps)
		if err != nil {
			return nil, err
		}
		orderedSteps = append(orderedSteps, steps...)
	}
	return orderedSteps, nil
} */
