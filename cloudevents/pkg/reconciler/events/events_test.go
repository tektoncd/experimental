package events_test

import (
	"fmt"
	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	rtesting "knative.dev/pkg/reconciler/testing"
	"regexp"
	"testing"
	"time"
)

func TestEmit(t *testing.T) {
	objectStatus := duckv1beta1.Status{
		Conditions: []apis.Condition{{
			Type:    apis.ConditionSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  v1beta1.PipelineRunReasonStarted.String(),
			Message: "just starting",
		}},
	}
	object := &v1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: "tekton.dev/v1beta",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test1",
			Namespace: "test",
		},
		Status: v1beta1.PipelineRunStatus{Status: objectStatus},
	}
	testcases := []struct {
		name           string
		data           map[string]string
		wantEvent      string
		wantCloudEvent string
	}{{
		name:           "without sink",
		data:           map[string]string{},
		wantEvent:      "Normal Started",
		wantCloudEvent: "",
	}, {
		name:           "with empty string sink",
		data:           map[string]string{"default-cloud-events-sink": ""},
		wantEvent:      "Normal Started",
		wantCloudEvent: "",
	}, {
		name:           "with sink",
		data:           map[string]string{"default-cloud-events-sink": "http://mysink"},
		wantEvent:      "Normal Started",
		wantCloudEvent: `(?s)dev.tekton.event.pipelinerun.started.v1.*test1`,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the context and seed test data
			ctx, _ := rtesting.SetupFakeContext(t)
			ctx = cloudevent.WithClient(ctx, &cloudevent.FakeClientBehaviour{SendSuccessfully: true})
			fakeClient := cloudevent.Get(ctx).(cloudevent.FakeClient)

			// Setup the config and add it to the context
			defaults, _ := config.NewDefaultsFromMap(tc.data)
			cfg := &config.CEConfig{
				Defaults: defaults,
			}
			ctx = config.ToContext(ctx, cfg)

			events.Emit(ctx, object)
			if err := checkCloudEvents(t, &fakeClient, tc.name, tc.wantCloudEvent); err != nil {
				t.Fatalf(err.Error())
			}
		})
	}
}

func eventFromChannel(c chan string, testName string, wantEvent string) error {
	timer := time.NewTimer(1 * time.Second)
	select {
	case event := <-c:
		if wantEvent == "" {
			return fmt.Errorf("received event \"%s\" for %s but none expected", event, testName)
		}
		matching, err := regexp.MatchString(wantEvent, event)
		if err == nil {
			if !matching {
				return fmt.Errorf("expected event \"%s\" but got \"%s\" instead for %s", wantEvent, event, testName)
			}
		}
	case <-timer.C:
		if wantEvent != "" {
			return fmt.Errorf("received no events for %s but %s expected", testName, wantEvent)
		}
	}
	return nil
}

func checkCloudEvents(t *testing.T, fce *cloudevent.FakeClient, testName string, wantEvent string) error {
	t.Helper()
	return eventFromChannel(fce.Events, testName, wantEvent)
}
