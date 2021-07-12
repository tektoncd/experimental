package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AddEventSentToCache adds the particular object to cache marking it as sent
func AddEventSentToCache(ctx context.Context, event *cloudevents.Event) error {
	cacheClient := Get(ctx)
	if cacheClient == nil {
		return errors.New("cache client is nil")
	}
	cacheClient.Add(eventKey(event), nil)
	return nil
}

// IsCloudEventSent checks if the event exists in the cache
func IsCloudEventSent(ctx context.Context, event *cloudevents.Event) (bool, error) {
	cacheClient := Get(ctx)
	if cacheClient == nil {
		return false, errors.New("cache client is nil")
	}
	return cacheClient.Contains(eventKey(event)), nil
}

// eventKey defines whether an event is considered different from another
// in future we might want to let specific event types override this
func eventKey(event *cloudevents.Event) string {
	var (
		data map[string]interface{}
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
