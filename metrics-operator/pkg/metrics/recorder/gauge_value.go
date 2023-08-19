package recorder

import (
	"fmt"
	"sync"

	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/util/sets"
)

type GaugeTagMapValue struct {
	tagMap       *tag.Map
	taskRunNames sets.Set[string]
}

type GaugeValue struct {
	m  map[string]GaugeTagMapValue
	rw sync.RWMutex
}

func (g *GaugeValue) ValueFor(tagMap *tag.Map) (float64, error) {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	tagMapValue, exists := g.m[tagMap.String()]
	if !exists {
		return 0.0, fmt.Errorf("tag does not exist")
	}

	return float64(tagMapValue.taskRunNames.Len()), nil
}

func (g *GaugeValue) Keys() []*tag.Map {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	result := []*tag.Map{}
	for _, tagMapValue := range g.m {
		result = append(result, tagMapValue.tagMap)
	}
	return result
}

func (g *GaugeValue) deleteTaskRunFromAllTagMapValues(taskRun *pipelinev1beta1.TaskRun, exceptions sets.Set[string]) {
	for key, tagMapValue := range g.m {
		if tagMapValue.taskRunNames.Has(taskRun.Name) && !exceptions.Has(key) {
			tagMapValue.taskRunNames = tagMapValue.taskRunNames.Delete(taskRun.Name)
		}
	}
}

func (g *GaugeValue) Delete(taskRun *pipelinev1beta1.TaskRun) {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	g.deleteTaskRunFromAllTagMapValues(taskRun, sets.New[string]())
}

func (g *GaugeValue) Update(taskRun *pipelinev1beta1.TaskRun, tagMap *tag.Map) {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	g.deleteTaskRunFromAllTagMapValues(taskRun, sets.New[string](tagMap.String()))

	// TODO: namespace
	tagMapValue, exists := g.m[tagMap.String()]
	if !exists {
		g.m[tagMap.String()] = GaugeTagMapValue{
			tagMap:       tagMap,
			taskRunNames: sets.New[string](taskRun.Name),
		}
	} else {
		tagMapValue.taskRunNames = tagMapValue.taskRunNames.Insert(taskRun.Name)
	}

}
