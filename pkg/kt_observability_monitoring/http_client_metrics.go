package kt_observability_monitoring

import (
	"sync"

	"github.com/keytiles/lib-errorhandling-golang/v2/pkg/kt_errors"
	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-utils-golang/v2/pkg/kt_utils"
	"github.com/prometheus/client_golang/prometheus"
)

// If you develop a client for a HTTP API you can use this class to quickly and efficiently attach Metrics to your client.
//
// This object is designed the way that it starts empty when created (has 0 Metric) and Metrics are getting created and exposed as you invoke it's methods. This
// is why it is "lazy". You can track how many times requests were sent, and how many times they succeeded / failed. You also have the possibility to track
// Request-Response loop times AND you can do it
// per each HttpSatatus codes if you want which brings pretty good observability just out of the box.
type HttpClientLazyMetricsSet struct {
	mu sync.Mutex

	of        string
	qualifier any
	clientId  string

	reqSentCounter *prometheus.Counter

	reqSuccessCounterByStatusCode map[string]prometheus.Counter
	reqProcessingTimeByStatusCode map[string]prometheus.Observer
	reqFailedCounterByStatusCode  map[string]prometheus.Counter
}

type HttpClientLazyMetricsSetOpt func(m *HttpClientLazyMetricsSet)

// Preferred constructor. Returns a Fault when 'of' is empty (mandatory endpoint name).
// On success the Fault is nil.
func NewHttpClientLazyMetricsSetOrFault(of string, opts ...HttpClientLazyMetricsSetOpt) (*HttpClientLazyMetricsSet, kt_errors.Fault) {
	methodName := "NewHttpClientLazyMetricsSetOrFault()"

	// 'of' is the endpoint name and must be provided by the caller.
	if of == "" {
		return nil, kt_errors.NewFaultBuilder(kt_errors.ValidationFault).
			WithErrorCodes(kt_errors.VALIDATION_ERRCODE_MISSING_MANDATORY).
			WithMessageTemplate("empty 'of' parameter is not allowed").
			WithSource(PACKAGE_NAME, methodName).
			Build()
	}

	metrics := HttpClientLazyMetricsSet{
		of:                            of,
		qualifier:                     "-",
		clientId:                      "-",
		reqSuccessCounterByStatusCode: make(map[string]prometheus.Counter),
		reqProcessingTimeByStatusCode: make(map[string]prometheus.Observer),
		reqFailedCounterByStatusCode:  make(map[string]prometheus.Counter),
	}

	for _, o := range opts {
		o(&metrics)
	}

	return &metrics, nil
}

// Creates a new metrics set you can use in your HTTP clients to create observability of invoking HTTP endpoints.
//
// Pass in "of" as the best name (meaningful) of the endpoint client is invoking! And feel free to use the optional setup too!
//
// Deprecated: use NewHttpClientLazyMetricsSetOrFault. On empty 'of' soft-fails (Warn + placeholder of "-"); does not panic.
func NewHttpClientLazyMetricsSet(of string, opts ...HttpClientLazyMetricsSetOpt) *HttpClientLazyMetricsSet {
	methodName := "NewHttpClientLazyMetricsSet()"

	metrics, fault := NewHttpClientLazyMetricsSetOrFault(of, opts...)
	if fault != nil {
		kt_logging.GetLogger(PACKAGE_NAME + ".HttpClientLazyMetricsSet").
			Warn("%v: soft-fail empty 'of' - using placeholder '-' - %s", methodName, kt_utils.VarPrinter{TheVar: fault})
		metrics, _ = NewHttpClientLazyMetricsSetOrFault("-", opts...)
	}
	return metrics
}

// Assigns a "qualifier" to all Metric instances in your set of your choice. One example of good qualifiers could be the httpMethod like GET, POST, PUT etc to
// reflect request type - but can be anything else too of course.
func WithHttpClientQualifier(qualifier any) HttpClientLazyMetricsSetOpt {
	return func(m *HttpClientLazyMetricsSet) {
		if qualifier != nil {
			m.qualifier = qualifier
		}
	}
}

// Deprecated: Use WithHttpClientQualifier() method from now!
func WithQualifier(qualifier any) HttpClientLazyMetricsSetOpt {
	return WithHttpClientQualifier(qualifier)
}

// Assigns a "clientId" to all Metric instances in your set. This is very useful if a specific client actually can have multiple instances for whatever reason.
func WithHttpClientId(id string) HttpClientLazyMetricsSetOpt {
	return func(m *HttpClientLazyMetricsSet) {
		if id != "" {
			m.clientId = id
		}
	}
}

// Deprecated: use WithHttpClientId() instead!
func WithClientId(id string) HttpClientLazyMetricsSetOpt {
	return WithHttpClientId(id)
}

// Invoke when client sent the request - will create+increase counter
func (m *HttpClientLazyMetricsSet) RequestSent() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.reqSentCounter == nil {
		c := GetCounterMetricInstance(
			GetClientRequestSentCountTemplate(),
			map[string]any{"of": m.of, "protocol": "http", "statusCode": "-", "qualifier": m.qualifier, "clientId": m.clientId},
		)
		m.reqSentCounter = &c
	}
	(*m.reqSentCounter).Inc()
}

// Invoke when client received a success - pass in the httpStatusCode what was returned. This will create+increase the appropriate success counter.
// The statusCode is taken as a string although normally it is int. Reason: this way if you do not want to distinguish fully just by ranges let's say you can
// send "2xx" to represent anything in 2xx range.
func (m *HttpClientLazyMetricsSet) RequestSucceeded(withHttpStatusCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, found := m.reqSuccessCounterByStatusCode[withHttpStatusCode]
	if !found {
		c = GetCounterMetricInstance(
			GetClientRequestSucceededCountTemplate(),
			map[string]any{"of": m.of, "protocol": "http", "statusCode": withHttpStatusCode, "qualifier": m.qualifier, "clientId": m.clientId},
		)
		m.reqSuccessCounterByStatusCode[withHttpStatusCode] = c
	}
	c.Inc()
}

// Invoke when client received a failure - pass in the httpStatusCode of the failure. This will create+increase the appropriate failure counter
// The statusCode is taken as a string although normally it is int. Reason: this way if you do not want to distinguish fully just by ranges let's say you can
// send "5xx" to represent anything in 5xx range.
func (m *HttpClientLazyMetricsSet) RequestFailed(withHttpStatusCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, found := m.reqFailedCounterByStatusCode[withHttpStatusCode]
	if !found {
		c = GetCounterMetricInstance(
			GetClientRequestFailedCountTemplate(),
			map[string]any{"of": m.of, "protocol": "http", "statusCode": withHttpStatusCode, "qualifier": m.qualifier, "clientId": m.clientId},
		)
		m.reqFailedCounterByStatusCode[withHttpStatusCode] = c
	}
	c.Inc()
}

// Track processing times - pass in the httpStatusCode so we can collect segregated. This will maintain a Summary
// The statusCode is taken as a string although normally it is int. Reason: this way if you do not want to distinguish fully just by ranges let's say you can
// send "2xx" to represent anything in 2xx range.
func (m *HttpClientLazyMetricsSet) RequestTookMillis(httpStatusCode string, millis float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	c, found := m.reqProcessingTimeByStatusCode[httpStatusCode]
	if !found {
		c = GetSummaryMetricInstance(
			GetClientRequestProcessingTimeTemplate(),
			map[string]any{"of": m.of, "protocol": "http", "statusCode": httpStatusCode, "qualifier": m.qualifier, "clientId": m.clientId},
		)
		m.reqProcessingTimeByStatusCode[httpStatusCode] = c
	}
	c.Observe(millis)
}
