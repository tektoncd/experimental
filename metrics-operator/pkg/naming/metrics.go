package naming

import (
	"fmt"
	"strings"
)

func CounterMetric(resource, monitorName, metricName string) string {
	return fmt.Sprintf("%s_%s_%s_total", resource, strings.ReplaceAll(monitorName, "-", "_"), metricName)
}

func HistogramMetric(resource, monitorName, metricName string) string {
	return fmt.Sprintf("%s_%s_%s_seconds", resource, strings.ReplaceAll(monitorName, "-", "_"), metricName)
}

func GaugeMetric(resource, monitorName, metricName string) string {
	return fmt.Sprintf("%s_%s_%s", resource, strings.ReplaceAll(monitorName, "-", "_"), metricName)
}

func MonitorId(resource, monitorName string) string {
	return fmt.Sprintf("%s/%s", resource, monitorName)
}
