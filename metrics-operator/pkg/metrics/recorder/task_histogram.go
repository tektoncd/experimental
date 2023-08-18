package recorder

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	monitoringv1alpha1 "github.com/tektoncd/experimental/metrics-operator/pkg/apis/monitoring/v1alpha1"
	"github.com/tektoncd/experimental/metrics-operator/pkg/naming"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/jsonpath"
	"knative.dev/pkg/logging"
)

func parseTime(field string, value reflect.Value) (*metav1.Time, error) {
	switch k := value.Interface().(type) {
	case *metav1.Time:
		return k.DeepCopy(), nil
	case metav1.Time:
		return k.DeepCopy(), nil
	case time.Time:
		return &metav1.Time{Time: k}, nil
	case *time.Time:
		if k == nil {
			return nil, nil
		}
		result := metav1.NewTime(*k)
		return &result, nil
	default:
		return nil, fmt.Errorf("could not parse '%s' duration, wrong type", field)
	}
}

// ParseDuration returns from, to and error
func ParseDuration(duration *monitoringv1alpha1.TaskMetricHistogramDuration, input any) (*metav1.Time, *metav1.Time, error) {
	j := jsonpath.New("duration")
	templateFrom := fmt.Sprintf("{%s}{%s}", duration.From, duration.To)
	err := j.Parse(templateFrom)

	if err != nil {
		return nil, nil, err
	}
	results, err := j.FindResults(input)
	if err != nil {
		return nil, nil, err
	}
	if len(results) != 2 {
		return nil, nil, fmt.Errorf("unable to parse duration, got %d results", len(results))
	}
	if len(results[0]) != 1 {
		return nil, nil, fmt.Errorf("unable to parse 'from' duration, got %d results", len(results[0]))
	}
	if len(results[1]) != 1 {
		return nil, nil, fmt.Errorf("unable to parse 'to' duration, got %d results", len(results[1]))
	}

	var from *metav1.Time
	var to *metav1.Time
	from, err = parseTime("from", results[0][0])
	if err != nil {
		return nil, nil, err
	}
	to, err = parseTime("to", results[1][0])
	if err != nil {
		return nil, nil, err
	}
	return from, to, nil

}

func ParseRFC3339(s string) (*metav1.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, err
	}
	metaTime := metav1.NewTime(t)
	return &metaTime, nil
}
func MustParseRFC3339(s string) *metav1.Time {
	metaTime, err := ParseRFC3339(s)
	if err != nil {
		panic(err)
	}
	return metaTime
}

type TaskHistogram struct {
	TaskName        string
	TaskMonitorName string
	TaskMetric      *v1alpha1.TaskMetric
	view            *view.View
	measure         *stats.Float64Measure
}

func (t *TaskHistogram) MetricName() string {
	return naming.HistogramMetric("task", t.TaskMonitorName, t.TaskMetric.Name)
}

func (t *TaskHistogram) MetricType() string {
	return "histogram"
}

func (t *TaskHistogram) MonitorName() string {
	return t.TaskMonitorName
}


func (t *TaskHistogram) View() *view.View {
	return t.view
}

func (t *TaskHistogram) Record(ctx context.Context, recorder stats.Recorder, taskRun *pipelinev1beta1.TaskRun) {
	logger := logging.FromContext(ctx).With("kind", "TaskMonitor", "monitor", t.TaskMonitorName, "metric", t.TaskMetric)
	if ref := taskRun.Spec.TaskRef; ref == nil || ref.Name != t.TaskName {
		return
	}
	tagMap, err := tagMapFromByStatements(t.TaskMetric.By, taskRun)
	if err != nil {
		logger.Errorw("error recording value, invalid tag map", zap.Error(err))
		return
	}

	from, to, err := ParseDuration(t.TaskMetric.Duration, taskRun)
	if err != nil {
		logger.Errorw("error parsing duration", zap.Error(err))
		return
	}
	if from == nil || to == nil {
		logger.Info("missing duration timestamp")
		return
	}
	duration := to.Sub(from.Time).Seconds()
	recorder.Record(tagMap, []stats.Measurement{t.measure.M(duration)}, map[string]any{})
}

func (t *TaskHistogram) Clean(ctx context.Context, taskRun *pipelinev1beta1.TaskRun) {
}

func NewTaskHistogram(metric *v1alpha1.TaskMetric, monitor *v1alpha1.TaskMonitor) *TaskHistogram {
	buckets := []float64{.25, .5, 1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000}
	histogram := &TaskHistogram{
		TaskName:        monitor.Spec.TaskName,
		TaskMonitorName: monitor.Name,
		TaskMetric:      metric,
	}
	histogram.measure = stats.Float64(histogram.MetricName(), fmt.Sprintf("histogram samples in seconds for TaskMonitor %s/%s", histogram.TaskMonitorName, histogram.TaskMetric.Name), stats.UnitSeconds)
	view := &view.View{
		Description: histogram.measure.Description(),
		Measure:     histogram.measure,
		Aggregation: view.Distribution(buckets...),
		TagKeys:     viewTags(metric.By),
	}
	histogram.view = view
	return histogram
}
