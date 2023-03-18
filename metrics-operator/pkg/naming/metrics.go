package naming

import (
	"fmt"
	"strings"
)

func CounterMetric(resource, monitorName, metricName string) string {
	return fmt.Sprintf("%s_%s_%s_total", resource, strings.ReplaceAll(monitorName, "-", "_"), metricName)
}
