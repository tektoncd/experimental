package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	lru "github.com/hashicorp/golang-lru"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

// AddEventSentToCache adds the particular object to cache marking it as sent
func AddEventSentToCache(cacheClient *lru.Cache, event *cloudevents.Event) error {
	if cacheClient == nil {
		return errors.New("cache client is nil")
	}
	cacheClient.Add(EventKey(event), nil)
	return nil
}

// IsCloudEventSent checks if the event exists in the cache
func IsCloudEventSent(cacheClient *lru.Cache, event *cloudevents.Event) (bool, error) {
	if cacheClient == nil {
		return false, errors.New("cache client is nil")
	}
	return cacheClient.Contains(EventKey(event)), nil
}

// eventKey defines whether an event is considered different from another
// in future we might want to let specific event types override this
func EventKey(event *cloudevents.Event) string {
	var (
		data              map[string]interface{}
		resourceType      string
		resourceName      string
		resourceNamespace string
	)
	json.Unmarshal(event.Data(), &data)
	for k, v := range data {
		resourceType = k
		switch k {
		case "taskrun":
			var run v1beta1.TaskRun
			json.Unmarshal([]byte(v.(string)), &run)
			resourceName = run.Name
			resourceNamespace = run.Namespace
		case "pipelinerun":
			var run v1beta1.PipelineRun
			json.Unmarshal([]byte(v.(string)), &run)
			resourceName = run.Name
			resourceNamespace = run.Namespace
		}
	}
	eventType := event.Type()
	return fmt.Sprintf("%s/%s/%s/%s", eventType, resourceType, resourceNamespace, resourceName)
}
