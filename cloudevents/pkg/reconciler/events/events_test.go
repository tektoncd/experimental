package events_test

import (
	"testing"

	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events"
	"github.com/tektoncd/experimental/cloudevents/pkg/reconciler/events/cloudevent"
	cetest "github.com/tektoncd/experimental/cloudevents/test"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
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
		wantCloudEvent []string
	}{{
		name:           "without sink",
		data:           map[string]string{},
		wantEvent:      "Normal Started",
		wantCloudEvent: []string{},
	}, {
		name:           "with empty string sink",
		data:           map[string]string{"default-cloud-events-sink": ""},
		wantEvent:      "Normal Started",
		wantCloudEvent: []string{},
	}, {
		name:           "with sink",
		data:           map[string]string{"default-cloud-events-sink": "http://mysink"},
		wantEvent:      "Normal Started",
		wantCloudEvent: []string{`(?s)cd.pipelinerun.queued.v1.*test1`},
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup the context and seed test data
			ctx, _ := cetest.SetupFakeContext(t)
			fakeClient := cloudevent.Get(ctx).(cloudevent.FakeClient)

			// Setup the config and add it to the context
			defaults, _ := config.NewDefaultsFromMap(tc.data)
			cfg := &config.CEConfig{
				Defaults: defaults,
			}
			ctx = config.ToContext(ctx, cfg)

			events.Emit(ctx, object)
			if err := cetest.CheckEventsUnordered(t, fakeClient.Events, tc.name, tc.wantCloudEvent); err != nil {
				t.Fatalf(err.Error())
			}
		})
	}
}
