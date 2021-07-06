package cache

import (
	"context"
	"errors"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// AddEventSentToCache adds the particular object to cache marking it as sent
func AddEventSentToCache(ctx context.Context, event *cloudevents.Event) error {
	cacheClient := Get(ctx)
	if cacheClient == nil {
		return errors.New("cache client is nil")
	}
	cacheClient.Add(event.String(), nil)
	return nil
}

// IsCloudEventSent checks if the event exists in the cache
func IsCloudEventSent(ctx context.Context, event *cloudevents.Event) (bool, error) {
	cacheClient := Get(ctx)
	if cacheClient == nil {
		return false, errors.New("cache client is nil")
	}
	return cacheClient.Contains(event.String()), nil
}
