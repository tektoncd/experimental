package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	lru "github.com/hashicorp/golang-lru"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		data         map[string]interface{}
		resourceType string
		resourceName string
	)
	json.Unmarshal(event.Data(), &data)
	for k, v := range data {
		if k == "pipelinerun" || k == "taskrun" {
			resourceType = k
			var meta metav1.ObjectMeta
			json.Unmarshal([]byte(v.(string)), &meta)
			resourceName = meta.Name
		}
	}
	eventType := event.Type()
	return fmt.Sprintf("%s/%s/%s", eventType, resourceType, resourceName)
}
