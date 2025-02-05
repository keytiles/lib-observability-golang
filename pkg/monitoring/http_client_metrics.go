package kt_observability_monitoring

import (
	"github.com/prometheus/client_golang/prometheus"
)

// If you develop a client for a HTTP API you can use this class to quickly and efficiently attach Metrics to your client.
//
// This object is designed the way that it starts empty when created (has 0 Metric) and Metrics are getting created and exposed as you invoke it's methods. This is why it is "lazy".
// You can track how many times requests were sent, and how many times they succeeded / failed. You also have the possibility to track Request-Response loop times AND you can do it
// per each HttpSatatus codes if you want which brings pretty good observability just out of the box.
type HttpClientLazyMetricsSet struct {
	of        string
	qualifier any
	clientId  string

	reqSentCounter *prometheus.Counter

	reqSuccessCounterByStatusCode map[int]prometheus.Counter
	reqProcessingTimeByStatusCode map[int]prometheus.Observer
	reqFailedCounterByStatusCode  map[int]prometheus.Counter
}

type HttpClientLazyMetricsSetOpt func(m *HttpClientLazyMetricsSet)

// Creates a new metrics set you can use in your HTTP clients to create observability of invoking HTTP endpoints.
//
// Pass in "of" as the best name (meaningful) of the endpoint client is invoking! And feel free to use the optional setup too!
func NewHttpClientLazyMetrics(of string, opts ...HttpClientLazyMetricsSetOpt) *HttpClientLazyMetricsSet {
	if of == "" {
		panic("Can not create HttpClientLazyMetrics with empty 'of' parameter!")
	}

	metrics := HttpClientLazyMetricsSet{
		of:                            of,
		qualifier:                     "-",
		clientId:                      "-",
		reqSuccessCounterByStatusCode: make(map[int]prometheus.Counter),
		reqProcessingTimeByStatusCode: make(map[int]prometheus.Observer),
		reqFailedCounterByStatusCode:  make(map[int]prometheus.Counter),
	}

	for _, o := range opts {
		o(&metrics)
	}

	return &metrics
}

// Assigns a "qualifier" to all Metric instances in your set of your choice. One example of good qualifiers could be the httpMethod like GET, POST, PUT etc to reflect request type - but can be anything else too of course.
func WithQualifier(qualifier any) HttpClientLazyMetricsSetOpt {
	return func(m *HttpClientLazyMetricsSet) {
		if qualifier != nil {
			m.qualifier = qualifier
		}
	}
}

// Assigns a "clientId" to all Metric instances in your set. This is very useful if a specific client actually can have multiple instances for whatever reason.
func WithClientId(id string) HttpClientLazyMetricsSetOpt {
	return func(m *HttpClientLazyMetricsSet) {
		if id != "" {
			m.clientId = id
		}
	}
}

// Invoke when client sent the request - will create+increase counter
func (m *HttpClientLazyMetricsSet) RequestSent() {
	if m.reqSentCounter == nil {
		c := GetCounterMetricInstance(GetClientRequestSentCountTemplate(), map[string]interface{}{"of": m.of, "protocol": "http", "statusCode": "-", "qualifier": m.qualifier, "clientId": m.clientId})
		m.reqSentCounter = &c
	}
	(*m.reqSentCounter).Inc()
}

// Invoke when client received a success - pass in the httpStatusCode what was returned. This will create+increase the appropriate success counter
func (m *HttpClientLazyMetricsSet) RequestSucceeded(withHttpStatusCode int) {
	c, found := m.reqSuccessCounterByStatusCode[withHttpStatusCode]
	if !found {
		c = GetCounterMetricInstance(GetClientRequestSucceededCountTemplate(), map[string]interface{}{"of": m.of, "protocol": "http", "statusCode": withHttpStatusCode, "qualifier": m.qualifier, "clientId": m.clientId})
		m.reqSuccessCounterByStatusCode[withHttpStatusCode] = c
	}
	c.Inc()
}

// Invoke when client received a failure - pass in the httpStatusCode of the failure. This will create+increase the appropriate failure counter
func (m *HttpClientLazyMetricsSet) RequestFailed(withHttpStatusCode int) {
	c, found := m.reqFailedCounterByStatusCode[withHttpStatusCode]
	if !found {
		c = GetCounterMetricInstance(GetClientRequestFailedCountTemplate(), map[string]interface{}{"of": m.of, "protocol": "http", "statusCode": withHttpStatusCode, "qualifier": m.qualifier, "clientId": m.clientId})
		m.reqFailedCounterByStatusCode[withHttpStatusCode] = c
	}
	c.Inc()
}

// Track processing times - pass in the httpStatusCode so we can collect segregated. This will maintain a Summary
func (m *HttpClientLazyMetricsSet) RequestTookMillis(httpStatusCode int, millis float64) {
	c, found := m.reqProcessingTimeByStatusCode[httpStatusCode]
	if !found {
		c = GetSummaryMetricInstance(GetClientRequestProcessingTimeTemplate(), map[string]interface{}{"of": m.of, "protocol": "http", "statusCode": httpStatusCode, "qualifier": m.qualifier, "clientId": m.clientId})
		m.reqProcessingTimeByStatusCode[httpStatusCode] = c
	}
	c.Observe(millis)
}
