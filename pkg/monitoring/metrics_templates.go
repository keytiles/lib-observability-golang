package kt_observability_monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	metricTemplatesAvailable bool

	// Generic execution counter - "of" something/anything
	execCount_template MetricTemplate
	// Generic error counter - "of" something/anything
	errorCount_template MetricTemplate
	// Generic warning counter - "of" something/anything
	warningCount_template MetricTemplate
	// Generic processing time (summary - observer) - "of" something/anything
	processingTime_template MetricTemplate

	// Generic "req sent" counter for synchronous request clients (HTTP, gRPC, etc)
	clientReqSentCount_template MetricTemplate
	// Generic "req success" counter for synchronous request clients (HTTP, gRPC, etc)
	clientReqSucceededCount_template MetricTemplate
	// Generic "req failed" counter for synchronous request clients (HTTP, gRPC, etc)
	clientReqFailedCount_template MetricTemplate
	// Generic "req retried" warning counter for synchronous request clients (HTTP, gRPC, etc)
	clientReqRetriedWarnCount_template MetricTemplate
	// Generic "req took time" (summary - observer) for synchronous request clients (HTTP, gRPC, etc)
	clientReqProcessingTime_template MetricTemplate
)

func createMetricTemplatesIfNotCreatedYet(reg prometheus.Registerer) {
	if metricTemplatesAvailable {
		// we have them already - skip
		return
	}
	metricTemplatesAvailable = true

	// "of" - you can add the name of the endpoint here you are invoking
	// "protocol" - protocol of your client, e.g. "http" or "grpc" or whatever
	// "statusCode" - makes sense for failure/retry maybe processing time cases? You can add here the statusCode you received from the server,
	//                e.g. in HTTP "400" or "500". For request sent counts makes no sense so just leave it empty ""
	// "qualifier" - anything else your use case finds useful - or leave empty ""
	customClientMetricsLabels := []string{"of", "protocol", "statusCode", "qualifier"}

	clientReqProcessingTime_template = GetSummaryMetricTemplate(
		prometheus.SummaryOpts{
			Namespace: "",
			Name:      "clientReqProcessingTime",
			Help:      "Reports processing time of a sync client request (check 'of' attribute!)",
		}, customClientMetricsLabels,
	)
	clientReqProcessingTime_template.Register(reg)

	clientReqSentCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "clientReqSentCount",
			Help:      "Reports count of a sync client request (check 'of' attribute!)",
		}, customClientMetricsLabels,
	)
	clientReqSentCount_template.Register(reg)

	clientReqSucceededCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "clientReqSuccessCount",
			Help:      "Reports success count of a sync client request (check 'of' attribute!)",
		}, customClientMetricsLabels,
	)
	clientReqSucceededCount_template.Register(reg)

	clientReqRetriedWarnCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "clientReqRetriedWarnCount",
			Help:      "Reports count of times a sync client request had to be retried (check 'of' attribute!)",
		}, customClientMetricsLabels,
	)
	clientReqRetriedWarnCount_template.Register(reg)

	clientReqFailedCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "clientReqFailedCount",
			Help:      "Reports failure count of a sync client request (check 'of' attribute!)",
		}, customClientMetricsLabels,
	)
	clientReqFailedCount_template.Register(reg)

	customGenericLabels := []string{"of", "qualifier"}

	processingTime_template = GetSummaryMetricTemplate(
		prometheus.SummaryOpts{
			Namespace: "",
			Name:      "processingTime",
			Help:      "Reports processing time of something (check 'of' attribute!)",
		}, customGenericLabels,
	)
	processingTime_template.Register(reg)

	execCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "execCount",
			Help:      "Reports count executions of something (check 'of' attribute!)",
		}, customGenericLabels,
	)
	execCount_template.Register(reg)

	errorCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "errorCount",
			Help:      "Reports count of a failure of something (check 'of' attribute!)",
		}, customGenericLabels,
	)
	errorCount_template.Register(reg)

	warningCount_template = GetCounterMetricTemplate(
		prometheus.CounterOpts{
			Namespace: "",
			Name:      "warningCount",
			Help:      "Reports count of a warning of something (check 'of' attribute!)",
		}, customGenericLabels,
	)
	warningCount_template.Register(reg)
}

// Returns a pre-defined template of a Counter which you can use to "count executions of something". Something which is part of your normal business logic. And you just want to be able to monitor it.
func GetExecCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return execCount_template
}

// Returns a pre-defined template of a Counter which you can use to "count of failures of something". Something which is part of your normal business logic. And you just want to be able to monitor it.
func GetErrorCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return errorCount_template
}

// Returns a pre-defined template of a Counter which you can use to "count of warnings of something". Something which is part of your normal business logic. And you just want to be able to monitor it.
func GetWarningCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return warningCount_template
}

// Returns a pre-defined template of a Counter which you can use to report "processing time of something". Something which is part of your normal business logic. And you just want to be able to monitor it.
func GetProcessingTimeTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return processingTime_template
}

// Returns a pre-defined template you can use in any synchronous clients (http, grpc, etc) to "count how many times a specific req is sent".
func GetClientRequestSentCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return clientReqSentCount_template
}

// Returns a pre-defined template you can use in any synchronous clients (http, grpc, etc) to "count how many times a specific req succeeded".
func GetClientRequestSucceededCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return clientReqSucceededCount_template
}

// Returns a pre-defined template you can use in any synchronous clients (http, grpc, etc) to "count how many times you had to retry a specific req".
func GetClientRequestRetriedWarnCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return clientReqRetriedWarnCount_template
}

// Returns a pre-defined template you can use in any synchronous clients (http, grpc, etc) to "count how many times a specific req has failed".
func GetClientRequestFailedCountTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return clientReqFailedCount_template
}

// Returns a pre-defined template you can use in any synchronous clients (http, grpc, etc) to report "how much time the specific req took".
func GetClientRequestProcessingTimeTemplate() MetricTemplate {
	createMetricTemplatesIfNotCreatedYet(MetricRegistry)
	return clientReqProcessingTime_template
}
