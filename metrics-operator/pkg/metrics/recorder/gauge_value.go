package recorder

import (
	"fmt"
	"sync"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"go.opencensus.io/tag"
	"k8s.io/apimachinery/pkg/util/sets"
)

type GaugeTagMapValue struct {
	tagMap *tag.Map
	runIds sets.Set[string]
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

	return float64(tagMapValue.runIds.Len()), nil
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

func (g *GaugeValue) deleteTaskRunFromAllTagMapValues(run *v1alpha1.RunDimensions, exceptions sets.Set[string]) {
	for key, tagMapValue := range g.m {
		if tagMapValue.runIds.Has(run.GetId()) && !exceptions.Has(key) {
			tagMapValue.runIds = tagMapValue.runIds.Delete(run.GetId())
		}
	}
}

func (g *GaugeValue) Delete(run *v1alpha1.RunDimensions) {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	g.deleteTaskRunFromAllTagMapValues(run, sets.New[string]())
}

func (g *GaugeValue) Update(run *v1alpha1.RunDimensions, tagMap *tag.Map) {
	g.rw.Lock()
	defer g.rw.Unlock()
	if g.m == nil {
		g.m = map[string]GaugeTagMapValue{}
	}

	g.deleteTaskRunFromAllTagMapValues(run, sets.New[string](tagMap.String()))

	tagMapValue, exists := g.m[tagMap.String()]
	if !exists {
		g.m[tagMap.String()] = GaugeTagMapValue{
			tagMap: tagMap,
			runIds: sets.New[string](run.GetId()),
		}
	} else {
		tagMapValue.runIds = tagMapValue.runIds.Insert(run.GetId())
	}

}
