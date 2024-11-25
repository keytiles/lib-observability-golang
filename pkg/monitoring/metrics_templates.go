package kt_observability_monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricTemplatesAvailable bool

	execCount_template      MetricTemplate
	errorCount_template     MetricTemplate
	warningCount_template   MetricTemplate
	processingTime_template MetricTemplate
)

func createMetricTemplatesIfNotCreatedYet(reg prometheus.Registerer) {
	if metricTemplatesAvailable {
		// we have them already - skip
		return
	}
	metricTemplatesAvailable = true

	customLabels := []string{"of", "qualifier"}

	processingTime_template = GetSummaryMetricTemplate(
		prometheus.SummaryOpts{
			Namespace: "",
			Name:      "processingTime",
			Help:      "Reports processing time of something (check 'of' attribute!)",
		}, customLabels,
	)
	processingTime_template.Register(reg)

	execCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "execCount",
			Help:      "Reports count executions of something (check 'of' attribute!)",
		}, customLabels,
	)
	execCount_template.Register(reg)

	errorCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "errorCount",
			Help:      "Reports count of a failure of something (check 'of' attribute!)",
		}, customLabels,
	)
	errorCount_template.Register(reg)

	warningCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "warningCount",
			Help:      "Reports count of a warning of something (check 'of' attribute!)",
		}, customLabels,
	)
	warningCount_template.Register(reg)
}

// Returns a pre-defined template of a Counter which you can use to "count executions of something". Something which is part of your normal business logic. And you just want to be able to monitor it.
//
// For example
func GetExecCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return execCount_template
}

func GetErrorCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return errorCount_template
}

func GetWarningCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return warningCount_template
}

func GetProcessingTimeTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return processingTime_template
}
