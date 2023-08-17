package recorder

import (
	"testing"

	monitoringv1alpha1 "github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseDuration(t *testing.T) {
	metric := &monitoringv1alpha1.TaskMetric{
		Type: "histogram",
		Name: "duration",
		Duration: &monitoringv1alpha1.TaskMetricHistogramDuration{
			From: ".metadata.creationTimestamp",
			To:   ".status.completionTime",
		},
	}

	taskRun := &pipelinev1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			CreationTimestamp: *MustParseRFC3339("2023-08-16T15:59:06Z"),
		},
		Status: pipelinev1beta1.TaskRunStatus{
			TaskRunStatusFields: pipelinev1beta1.TaskRunStatusFields{
				StartTime:      MustParseRFC3339("2023-08-16T15:59:26Z"),
				CompletionTime: MustParseRFC3339("2023-08-16T15:59:36Z"),
			},
		},
	}

	from, to, err := ParseDuration(metric.Duration, taskRun)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("from: %+v", from)
	t.Logf("to: %+v", to)
	t.Logf("duration: %+v", to.Sub(from.Time).Seconds())
	duration := to.Sub(from.Time).Seconds()
	if duration != 30 {
		t.Errorf("expected 30s, but got %fs", duration)
	}
}
