package kt_observability_monitoring

import (
	"net/http"
	"sync"

	"github.com/keytiles/lib-errorhandling-golang/v2/pkg/kt_errors"
	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-utils-golang/v2/pkg/kt_utils"
	"github.com/prometheus/client_golang/prometheus"
)

// If you develop a server for a HTTP API you can use this class to quickly and efficiently attach Metrics to your server.
//
// This object is designed the way that it starts empty when created (has 0 Metric) and Metrics are getting created and exposed as you invoke it's methods. This
// is why it is "lazy". You can track how many times requests were sent, and how many times they succeeded / failed. You also have the possibility to track
// Request-Response loop times AND you can do it
// per each HttpSatatus codes/ Methods which brings pretty good observability just out of the box.
type HttpServerLazyMetricsSet struct {
	mu sync.Mutex

	of       string
	serverId string

	serveStartedCounter             map[string]prometheus.Counter
	serveSuccessCounterByStatusCode map[string]prometheus.Counter
	serveProcessingTimeByStatusCode map[string]prometheus.Observer
	serveFailedCounterByStatusCode  map[string]prometheus.Counter
}

type HttpServerLazyMetricsSetOpt func(m *HttpServerLazyMetricsSet)

// Preferred constructor. Returns a Fault when 'of' is empty (mandatory endpoint / handler name).
// On success the Fault is nil.
func NewHttpServerLazyMetricsSetOrFault(of string, opts ...HttpServerLazyMetricsSetOpt) (*HttpServerLazyMetricsSet, kt_errors.Fault) {
	methodName := "NewHttpServerLazyMetricsSetOrFault()"

	// 'of' is the endpoint / handler name and must be provided by the caller.
	if of == "" {
		return nil, kt_errors.NewFaultBuilder(kt_errors.ValidationFault).
			WithErrorCodes(kt_errors.VALIDATION_ERRCODE_MISSING_MANDATORY).
			WithMessageTemplate("empty 'of' parameter is not allowed").
			WithSource(PACKAGE_NAME, methodName).
			Build()
	}

	metrics := HttpServerLazyMetricsSet{
		of:                              of,
		serverId:                        "-",
		serveStartedCounter:             make(map[string]prometheus.Counter),
		serveSuccessCounterByStatusCode: make(map[string]prometheus.Counter),
		serveProcessingTimeByStatusCode: make(map[string]prometheus.Observer),
		serveFailedCounterByStatusCode:  make(map[string]prometheus.Counter),
	}

	for _, o := range opts {
		o(&metrics)
	}

	return &metrics, nil
}

// Creates a new metrics set you can use in your HTTP servers to create observability of serving HTTP endpoints.
//
// Pass in "of" as the best name (meaningful) of the HTTP server/handler is invoking! And feel free to use the optional setup too!
//
// Deprecated: use NewHttpServerLazyMetricsSetOrFault. On empty 'of' soft-fails (Warn + placeholder of "-"); does not panic.
func NewHttpServerLazyMetricsSet(of string, opts ...HttpServerLazyMetricsSetOpt) *HttpServerLazyMetricsSet {
	methodName := "NewHttpServerLazyMetricsSet()"

	metrics, fault := NewHttpServerLazyMetricsSetOrFault(of, opts...)
	if fault != nil {
		kt_logging.GetLogger(PACKAGE_NAME + ".HttpServerLazyMetricsSet").
			Warn("%v: soft-fail empty 'of' - using placeholder '-' - %s", methodName, kt_utils.VarPrinter{TheVar: fault})
		metrics, _ = NewHttpServerLazyMetricsSetOrFault("-", opts...)
	}
	return metrics
}

// Assigns a "serverId" to all Metric instances in your set. This is very useful if a specific client actually can have multiple instances for whatever reason.
func WithHttpServerId(id string) HttpServerLazyMetricsSetOpt {
	return func(m *HttpServerLazyMetricsSet) {
		if id != "" {
			m.serverId = id
		}
	}
}

func getReqMethod(req *http.Request) string {
	if req == nil {
		return "-"
	}
	return req.Method
}

// Invoke when server started to process the request - will create+increase counter
func (m *HttpServerLazyMetricsSet) ServeStarted(req *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	method := getReqMethod(req)
	c, found := m.serveStartedCounter[method]
	if !found {
		c = GetCounterMetricInstance(
			GetServerServeStartedCountTemplate(),
			map[string]any{"of": m.of, "protocol": "http", "statusCode": "-", "qualifier": method, "serverId": m.serverId},
		)
		m.serveStartedCounter[method] = c
	}
	c.Inc()
}

// Invoke when server successfully served the request - pass in the httpStatusCode was returned to client. This will create+increase the appropriate success
// counter. The statusCode is taken as a string although normally it is int. Reason: this way if you do not want to distinguish fully just by ranges let's say
// you can send "2xx" to represent anything in 2xx range.
func (m *HttpServerLazyMetricsSet) ServeSucceeded(req *http.Request, withHttpStatusCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	method := getReqMethod(req)
	key := method + withHttpStatusCode
	c, found := m.serveSuccessCounterByStatusCode[key]
	if !found {
		c = GetCounterMetricInstance(
			GetServerServeSucceededCountTemplate(),
			map[string]any{"of": m.of, "protocol": "http", "statusCode": withHttpStatusCode, "qualifier": method, "serverId": m.serverId},
		)
		m.serveSuccessCounterByStatusCode[key] = c
	}
	c.Inc()
}

// Invoke when server failed to server the request - pass in the httpStatusCode was returned to client. This will create+increase the appropriate failure
// counter The statusCode is taken as a string although normally it is int. Reason: this way if you do not want to distinguish fully just by ranges let's say
// you can send "5xx" to represent anything in 5xx range.
func (m *HttpServerLazyMetricsSet) ServeFailed(req *http.Request, withHttpStatusCode string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	method := getReqMethod(req)
	key := method + withHttpStatusCode
	c, found := m.serveFailedCounterByStatusCode[key]
	if !found {
		c = GetCounterMetricInstance(
			GetServerServeFailedCountTemplate(),
			map[string]any{"of": m.of, "protocol": "http", "statusCode": withHttpStatusCode, "qualifier": method, "serverId": m.serverId},
		)
		m.serveFailedCounterByStatusCode[key] = c
	}
	c.Inc()
}

// Track processing times of serving the request - pass in the httpStatusCode so we can collect segregated. This will maintain a Summary.
// The statusCode is taken as a string although normally it is int. Reason: this way if you do not want to distinguish fully just by ranges let's say you can
// send "2xx" to represent anything in 2xx range.
func (m *HttpServerLazyMetricsSet) ServeTookMillis(req *http.Request, withHttpStatusCode string, millis float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	method := getReqMethod(req)
	key := method + withHttpStatusCode
	c, found := m.serveProcessingTimeByStatusCode[key]
	if !found {
		c = GetSummaryMetricInstance(
			GetServerServeProcessingTimeTemplate(),
			map[string]any{"of": m.of, "protocol": "http", "statusCode": withHttpStatusCode, "qualifier": method, "serverId": m.serverId},
		)
		m.serveProcessingTimeByStatusCode[key] = c
	}
	c.Observe(millis)
}
