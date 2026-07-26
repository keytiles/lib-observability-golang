package kt_observability_monitoring_test

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/keytiles/lib-errorhandling-golang/v2/pkg/kt_errors"
	"github.com/keytiles/lib-observability-golang/v2/pkg/kt_observability_monitoring"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

var initMetricsOnce sync.Once

// Happy-path: init registry, create template instances, record values, gather and assert series.
func TestMetricsHappyPath_TemplatesAndGather(t *testing.T) {
	// GIVEN
	ensureMetricsInitialized(t)

	exec := kt_observability_monitoring.GetCounterMetricInstance(
		kt_observability_monitoring.GetExecCountTemplate(),
		map[string]any{"of": "unitWork", "qualifier": "happy-path"},
	)
	errCount := kt_observability_monitoring.GetCounterMetricInstance(
		kt_observability_monitoring.GetErrorCountTemplate(),
		map[string]any{"of": "unitWork", "qualifier": "happy-path"},
	)
	procTime := kt_observability_monitoring.GetSummaryMetricInstance(
		kt_observability_monitoring.GetProcessingTimeTemplate(),
		map[string]any{"of": "unitWork", "qualifier": "happy-path"},
	)

	// WHEN
	exec.Inc()
	exec.Inc()
	errCount.Inc()
	procTime.Observe(12.5)

	families, err := kt_observability_monitoring.MetricRegistry.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}

	// THEN
	execMetric := findMetricFamily(t, families, "execCount")
	assertCounterValue(t, execMetric, map[string]string{
		"of": "unitWork", "qualifier": "happy-path", "metricType": "counter",
		"serviceName": "obs-test",
	}, 2)

	errorMetric := findMetricFamily(t, families, "errorCount")
	assertCounterValue(t, errorMetric, map[string]string{
		"of": "unitWork", "qualifier": "happy-path", "metricType": "counter",
		"serviceName": "obs-test",
	}, 1)

	processingMetric := findMetricFamily(t, families, "processingTime")
	assertSummarySampleCount(t, processingMetric, map[string]string{
		"of": "unitWork", "qualifier": "happy-path", "metricType": "summary",
		"serviceName": "obs-test",
	}, 1)
}

// Happy-path: HttpServerLazyMetricsSet creates and updates server serve metrics.
func TestHttpServerLazyMetricsSet_HappyPath(t *testing.T) {
	// GIVEN — registry/templates already initialized by previous test or init here safely
	ensureMetricsInitialized(t)

	metrics := kt_observability_monitoring.NewHttpServerLazyMetricsSet(
		"ping",
		kt_observability_monitoring.WithHttpServerId("srv-1"),
	)
	req, err := http.NewRequest(http.MethodGet, "http://example.local/api/v1/ping", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}

	// WHEN
	metrics.ServeStarted(req)
	metrics.ServeSucceeded(req, "200")
	metrics.ServeTookMillis(req, "200", 42)

	families, err := kt_observability_monitoring.MetricRegistry.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}

	// THEN
	started := findMetricFamily(t, families, "serverServeStartedCount")
	assertCounterValue(t, started, map[string]string{
		"of": "ping", "protocol": "http", "qualifier": "GET", "serverId": "srv-1", "metricType": "counter",
	}, 1)

	succeeded := findMetricFamily(t, families, "serverServeSuccessCount")
	assertCounterValue(t, succeeded, map[string]string{
		"of": "ping", "protocol": "http", "statusCode": "200", "qualifier": "GET", "serverId": "srv-1", "metricType": "counter",
	}, 1)

	took := findMetricFamily(t, families, "serverServeProcessingTime")
	assertSummarySampleCount(t, took, map[string]string{
		"of": "ping", "protocol": "http", "statusCode": "200", "qualifier": "GET", "serverId": "srv-1", "metricType": "summary",
	}, 1)
}

// Nil customLabels must not panic when creating a Counter instance (only metricType label needed).
func TestGetCounterMetricInstance_NilCustomLabels_DoesNotPanic(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	tpl := kt_observability_monitoring.GetCounterMetricTemplate(
		prometheus.CounterOpts{Name: "nilCustomLabelsCounter", Help: "test nil customLabels"},
		[]string{},
	)
	tpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	var counter prometheus.Counter
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic with nil customLabels, got: %v", r)
			}
		}()
		counter = kt_observability_monitoring.GetCounterMetricInstance(tpl, nil)
	}()

	// ---- THEN
	if counter == nil {
		t.Fatal("expected non-nil counter instance")
	}
	counter.Inc()
}

// Nil customLabels must not panic when creating a Gauge instance (only metricType label needed).
func TestGetGaugeMetricInstance_NilCustomLabels_DoesNotPanic(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	tpl := kt_observability_monitoring.GetGaugeMetricTemplate(
		prometheus.GaugeOpts{Name: "nilCustomLabelsGauge", Help: "test nil customLabels"},
		[]string{},
	)
	tpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	var gauge prometheus.Gauge
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic with nil customLabels, got: %v", r)
			}
		}()
		gauge = kt_observability_monitoring.GetGaugeMetricInstance(tpl, nil)
	}()

	// ---- THEN
	if gauge == nil {
		t.Fatal("expected non-nil gauge instance")
	}
	gauge.Set(1)
}

// Nil customLabels must not panic when creating a Summary instance (only metricType label needed).
func TestGetSummaryMetricInstance_NilCustomLabels_DoesNotPanic(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	tpl := kt_observability_monitoring.GetSummaryMetricTemplate(
		prometheus.SummaryOpts{Name: "nilCustomLabelsSummary", Help: "test nil customLabels"},
		[]string{},
	)
	tpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	var observer prometheus.Observer
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic with nil customLabels, got: %v", r)
			}
		}()
		observer = kt_observability_monitoring.GetSummaryMetricInstance(tpl, nil)
	}()

	// ---- THEN
	if observer == nil {
		t.Fatal("expected non-nil summary observer instance")
	}
	observer.Observe(1.5)
}

// Register with a nil Registerer interface must not panic.
func TestMetricTemplate_Register_NilInterface_DoesNotPanic(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	tpl := kt_observability_monitoring.GetCounterMetricTemplate(
		prometheus.CounterOpts{Name: "nilRegistererCounter", Help: "test nil registerer"},
		[]string{},
	)

	// ---- WHEN
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic on Register(nil), got: %v", r)
			}
		}()
		var reg prometheus.Registerer // nil interface
		tpl.Register(reg)
	}()

	// ---- THEN
	if tpl.IsRegistered() {
		t.Fatal("expected template to remain unregistered after Register(nil)")
	}
}

// Overlapping global ConstLabel and variable label names must not panic New*Vec.
func TestGetCounterMetricTemplate_GlobalLabelOverlap_DoesNotPanic(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	prev := kt_observability_monitoring.GetGlobalLabels()
	t.Cleanup(func() {
		kt_observability_monitoring.SetGlobalLabels(prev)
	})
	kt_observability_monitoring.SetGlobalLabels(map[string]any{
		"serviceName": "obs-test",
		"of":          "clash", // overlaps variable label "of"
	})

	// ---- WHEN
	var tpl kt_observability_monitoring.MetricTemplate
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic on const/variable label overlap, got: %v", r)
			}
		}()
		tpl = kt_observability_monitoring.GetCounterMetricTemplate(
			prometheus.CounterOpts{Name: "overlapLabelsCounter", Help: "test overlap"},
			[]string{"of"},
		)
	}()

	// ---- THEN — template usable without crash (Register + instance soft-fail ok)
	tpl.Register(kt_observability_monitoring.MetricRegistry)
	c := kt_observability_monitoring.GetCounterMetricInstance(tpl, map[string]any{"of": "work"})
	if c == nil {
		t.Fatal("expected non-nil counter from soft-fail template")
	}
	c.Inc()
}

// Incomplete custom labels must not panic the process; returned metric stays usable (soft-fail / discarded).
func TestGetCounterMetricInstance_MissingLabels_DoesNotPanic(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	tpl := kt_observability_monitoring.GetCounterMetricTemplate(
		prometheus.CounterOpts{Name: "missingLabelsCounter", Help: "test missing labels"},
		[]string{"of"},
	)
	tpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	var counter prometheus.Counter
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic with missing custom labels, got: %v", r)
			}
		}()
		// metricType is added by getter; "of" is intentionally missing
		counter = kt_observability_monitoring.GetCounterMetricInstance(tpl, map[string]any{})
	}()

	// ---- THEN
	if counter == nil {
		t.Fatal("expected non-nil counter (discarded/soft-fail is ok)")
	}
	counter.Inc()
}

// Extra unexpected custom labels must not panic the process.
func TestGetCounterMetricInstance_ExtraLabels_DoesNotPanic(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	tpl := kt_observability_monitoring.GetCounterMetricTemplate(
		prometheus.CounterOpts{Name: "extraLabelsCounter", Help: "test extra labels"},
		[]string{"of"},
	)
	tpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	var counter prometheus.Counter
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic with extra custom labels, got: %v", r)
			}
		}()
		counter = kt_observability_monitoring.GetCounterMetricInstance(tpl, map[string]any{
			"of": "work",
			"unexpected": "x",
		})
	}()

	// ---- THEN
	if counter == nil {
		t.Fatal("expected non-nil counter (discarded/soft-fail is ok)")
	}
	counter.Inc()
}

// Wrong template type into deprecated GetCounterMetricInstance must soft-fail (no panic; discarded counter usable).
func TestGetCounterMetricInstance_WrongTemplateType_DoesNotPanic(t *testing.T) {
	// ---- GIVEN — a Summary template passed where a Counter is required
	ensureMetricsInitialized(t)
	summaryTpl := kt_observability_monitoring.GetSummaryMetricTemplate(
		prometheus.SummaryOpts{Name: "wrongTypeForCounterDeprecated", Help: "summary used as counter (misuse)"},
		[]string{"of"},
	)
	summaryTpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	var counter prometheus.Counter
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic on wrong template type, got: %v", r)
			}
		}()
		counter = kt_observability_monitoring.GetCounterMetricInstance(summaryTpl, map[string]any{"of": "work"})
	}()

	// ---- THEN — discarded / soft-fail sink is ok; must remain usable
	if counter == nil {
		t.Fatal("expected non-nil counter (discarded/soft-fail is ok)")
	}
	counter.Inc()
}

// Wrong template type into GetCounterMetricInstanceOrFault must return a ValidationFault (wrong datatype); counter nil.
func TestGetCounterMetricInstanceOrFault_WrongTemplateType_ReturnsFault(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	summaryTpl := kt_observability_monitoring.GetSummaryMetricTemplate(
		prometheus.SummaryOpts{Name: "wrongTypeForCounterOrFault", Help: "summary used as counter (misuse)"},
		[]string{"of"},
	)
	summaryTpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	counter, fault := kt_observability_monitoring.GetCounterMetricInstanceOrFault(summaryTpl, map[string]any{"of": "work"})

	// ---- THEN
	if counter != nil {
		t.Fatal("expected nil counter when OrFault reports misuse")
	}
	if fault == nil {
		t.Fatal("expected non-nil Fault for wrong template type")
	}
	if fault.GetKind() != kt_errors.ValidationFault {
		t.Errorf("expected kind %q, got %q", kt_errors.ValidationFault, fault.GetKind())
	}
	if !fault.HasErrorCode(kt_errors.VALIDATION_ERRCODE_WRONG_DATATYPE) {
		t.Errorf("expected error code %q, got codes %v", kt_errors.VALIDATION_ERRCODE_WRONG_DATATYPE, fault.GetErrorCodes())
	}
}

// Wrong template type into deprecated GetSummaryMetricInstance must soft-fail (no panic; discarded observer usable).
func TestGetSummaryMetricInstance_WrongTemplateType_DoesNotPanic(t *testing.T) {
	// ---- GIVEN — a Counter template passed where a Summary is required
	ensureMetricsInitialized(t)
	counterTpl := kt_observability_monitoring.GetCounterMetricTemplate(
		prometheus.CounterOpts{Name: "wrongTypeForSummaryDeprecated", Help: "counter used as summary (misuse)"},
		[]string{"of"},
	)
	counterTpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	var observer prometheus.Observer
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic on wrong template type, got: %v", r)
			}
		}()
		observer = kt_observability_monitoring.GetSummaryMetricInstance(counterTpl, map[string]any{"of": "work"})
	}()

	// ---- THEN
	if observer == nil {
		t.Fatal("expected non-nil observer (discarded/soft-fail is ok)")
	}
	observer.Observe(1)
}

// Wrong template type into GetSummaryMetricInstanceOrFault must return a ValidationFault (wrong datatype); observer nil.
func TestGetSummaryMetricInstanceOrFault_WrongTemplateType_ReturnsFault(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	counterTpl := kt_observability_monitoring.GetCounterMetricTemplate(
		prometheus.CounterOpts{Name: "wrongTypeForSummaryOrFault", Help: "counter used as summary (misuse)"},
		[]string{"of"},
	)
	counterTpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	observer, fault := kt_observability_monitoring.GetSummaryMetricInstanceOrFault(counterTpl, map[string]any{"of": "work"})

	// ---- THEN
	if observer != nil {
		t.Fatal("expected nil observer when OrFault reports misuse")
	}
	if fault == nil {
		t.Fatal("expected non-nil Fault for wrong template type")
	}
	if fault.GetKind() != kt_errors.ValidationFault {
		t.Errorf("expected kind %q, got %q", kt_errors.ValidationFault, fault.GetKind())
	}
	if !fault.HasErrorCode(kt_errors.VALIDATION_ERRCODE_WRONG_DATATYPE) {
		t.Errorf("expected error code %q, got codes %v", kt_errors.VALIDATION_ERRCODE_WRONG_DATATYPE, fault.GetErrorCodes())
	}
}

// Wrong template type into deprecated GetGaugeMetricInstance must soft-fail (no panic; discarded gauge usable).
func TestGetGaugeMetricInstance_WrongTemplateType_DoesNotPanic(t *testing.T) {
	// ---- GIVEN — a Counter template passed where a Gauge is required
	ensureMetricsInitialized(t)
	counterTpl := kt_observability_monitoring.GetCounterMetricTemplate(
		prometheus.CounterOpts{Name: "wrongTypeForGaugeDeprecated", Help: "counter used as gauge (misuse)"},
		[]string{"of"},
	)
	counterTpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	var gauge prometheus.Gauge
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic on wrong template type, got: %v", r)
			}
		}()
		gauge = kt_observability_monitoring.GetGaugeMetricInstance(counterTpl, map[string]any{"of": "work"})
	}()

	// ---- THEN
	if gauge == nil {
		t.Fatal("expected non-nil gauge (discarded/soft-fail is ok)")
	}
	gauge.Set(1)
}

// Wrong template type into GetGaugeMetricInstanceOrFault must return a ValidationFault (wrong datatype); gauge nil.
func TestGetGaugeMetricInstanceOrFault_WrongTemplateType_ReturnsFault(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	counterTpl := kt_observability_monitoring.GetCounterMetricTemplate(
		prometheus.CounterOpts{Name: "wrongTypeForGaugeOrFault", Help: "counter used as gauge (misuse)"},
		[]string{"of"},
	)
	counterTpl.Register(kt_observability_monitoring.MetricRegistry)

	// ---- WHEN
	gauge, fault := kt_observability_monitoring.GetGaugeMetricInstanceOrFault(counterTpl, map[string]any{"of": "work"})

	// ---- THEN
	if gauge != nil {
		t.Fatal("expected nil gauge when OrFault reports misuse")
	}
	if fault == nil {
		t.Fatal("expected non-nil Fault for wrong template type")
	}
	if fault.GetKind() != kt_errors.ValidationFault {
		t.Errorf("expected kind %q, got %q", kt_errors.ValidationFault, fault.GetKind())
	}
	if !fault.HasErrorCode(kt_errors.VALIDATION_ERRCODE_WRONG_DATATYPE) {
		t.Errorf("expected error code %q, got codes %v", kt_errors.VALIDATION_ERRCODE_WRONG_DATATYPE, fault.GetErrorCodes())
	}
}

// Zero-value MetricTemplate must not nil-deref when Register tries to Warn (unknown metric type path).
func TestMetricTemplate_Register_ZeroValue_DoesNotNilDerefLogger(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	tpl := kt_observability_monitoring.MetricTemplate{}

	// ---- WHEN / THEN
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic/nil-deref on zero-value Register, got: %v", r)
			}
		}()
		tpl.Register(kt_observability_monitoring.MetricRegistry)
	}()
	if tpl.IsRegistered() {
		t.Fatal("expected zero-value template to remain unregistered")
	}
}

// Happy-path: HttpClientLazyMetricsSet creates and updates client request metrics.
func TestHttpClientLazyMetricsSet_HappyPath(t *testing.T) {
	// GIVEN
	ensureMetricsInitialized(t)

	metrics := kt_observability_monitoring.NewHttpClientLazyMetricsSet(
		"downstream",
		kt_observability_monitoring.WithHttpClientId("client-1"),
		kt_observability_monitoring.WithHttpClientQualifier("GET"),
	)

	// WHEN
	metrics.RequestSent()
	metrics.RequestSucceeded("200")
	metrics.RequestTookMillis("200", 33)

	families, err := kt_observability_monitoring.MetricRegistry.Gather()
	if err != nil {
		t.Fatalf("Gather failed: %v", err)
	}

	// THEN
	sent := findMetricFamily(t, families, "clientReqSentCount")
	assertCounterValue(t, sent, map[string]string{
		"of": "downstream", "protocol": "http", "qualifier": "GET", "clientId": "client-1", "metricType": "counter",
	}, 1)

	succeeded := findMetricFamily(t, families, "clientReqSuccessCount")
	assertCounterValue(t, succeeded, map[string]string{
		"of": "downstream", "protocol": "http", "statusCode": "200", "qualifier": "GET", "clientId": "client-1", "metricType": "counter",
	}, 1)

	took := findMetricFamily(t, families, "clientReqProcessingTime")
	assertSummarySampleCount(t, took, map[string]string{
		"of": "downstream", "protocol": "http", "statusCode": "200", "qualifier": "GET", "clientId": "client-1", "metricType": "summary",
	}, 1)
}

// Empty 'of' into deprecated NewHttpClientLazyMetricsSet must soft-fail (no panic; usable set with placeholder of).
func TestNewHttpClientLazyMetricsSet_EmptyOf_DoesNotPanic(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)

	// ---- WHEN
	var metrics *kt_observability_monitoring.HttpClientLazyMetricsSet
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic with empty of, got: %v", r)
			}
		}()
		metrics = kt_observability_monitoring.NewHttpClientLazyMetricsSet("")
	}()

	// ---- THEN — soft-fail set must remain usable (placeholder of is ok)
	if metrics == nil {
		t.Fatal("expected non-nil metrics set (soft-fail placeholder is ok)")
	}
	metrics.RequestSent()
}

// Empty 'of' into NewHttpClientLazyMetricsSetOrFault must return ValidationFault (missing mandatory); set nil.
func TestNewHttpClientLazyMetricsSetOrFault_EmptyOf_ReturnsFault(t *testing.T) {
	// ---- GIVEN / WHEN
	metrics, fault := kt_observability_monitoring.NewHttpClientLazyMetricsSetOrFault("")

	// ---- THEN
	if metrics != nil {
		t.Fatal("expected nil metrics set when OrFault reports empty of")
	}
	if fault == nil {
		t.Fatal("expected non-nil Fault for empty of")
	}
	if fault.GetKind() != kt_errors.ValidationFault {
		t.Errorf("expected kind %q, got %q", kt_errors.ValidationFault, fault.GetKind())
	}
	if !fault.HasErrorCode(kt_errors.VALIDATION_ERRCODE_MISSING_MANDATORY) {
		t.Errorf("expected error code %q, got codes %v", kt_errors.VALIDATION_ERRCODE_MISSING_MANDATORY, fault.GetErrorCodes())
	}
}

// Empty 'of' into deprecated NewHttpServerLazyMetricsSet must soft-fail (no panic; usable set with placeholder of).
func TestNewHttpServerLazyMetricsSet_EmptyOf_DoesNotPanic(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)

	// ---- WHEN
	var metrics *kt_observability_monitoring.HttpServerLazyMetricsSet
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("expected no panic with empty of, got: %v", r)
			}
		}()
		metrics = kt_observability_monitoring.NewHttpServerLazyMetricsSet("")
	}()

	// ---- THEN
	if metrics == nil {
		t.Fatal("expected non-nil metrics set (soft-fail placeholder is ok)")
	}
	req, err := http.NewRequest(http.MethodGet, "http://example.local/", nil)
	if err != nil {
		t.Fatalf("NewRequest failed: %v", err)
	}
	metrics.ServeStarted(req)
}

// Empty 'of' into NewHttpServerLazyMetricsSetOrFault must return ValidationFault (missing mandatory); set nil.
func TestNewHttpServerLazyMetricsSetOrFault_EmptyOf_ReturnsFault(t *testing.T) {
	// ---- GIVEN / WHEN
	metrics, fault := kt_observability_monitoring.NewHttpServerLazyMetricsSetOrFault("")

	// ---- THEN
	if metrics != nil {
		t.Fatal("expected nil metrics set when OrFault reports empty of")
	}
	if fault == nil {
		t.Fatal("expected non-nil Fault for empty of")
	}
	if fault.GetKind() != kt_errors.ValidationFault {
		t.Errorf("expected kind %q, got %q", kt_errors.ValidationFault, fault.GetKind())
	}
	if !fault.HasErrorCode(kt_errors.VALIDATION_ERRCODE_MISSING_MANDATORY) {
		t.Errorf("expected error code %q, got codes %v", kt_errors.VALIDATION_ERRCODE_MISSING_MANDATORY, fault.GetErrorCodes())
	}
}

// Parallel use of one HttpClientLazyMetricsSet must be race-free (and must not panic).
func TestHttpClientLazyMetricsSet_ConcurrentUse_NoRace(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	// Warm templates so concurrent stress targets the lazy-set maps, not first-time template init.
	_ = kt_observability_monitoring.GetClientRequestSentCountTemplate()
	_ = kt_observability_monitoring.GetClientRequestSucceededCountTemplate()
	_ = kt_observability_monitoring.GetClientRequestFailedCountTemplate()
	_ = kt_observability_monitoring.GetClientRequestProcessingTimeTemplate()

	metrics := kt_observability_monitoring.NewHttpClientLazyMetricsSet(
		"concurrent-client",
		kt_observability_monitoring.WithHttpClientId("c-race"),
		kt_observability_monitoring.WithHttpClientQualifier("GET"),
	)
	const goroutines = 32
	const iters = 40

	// ---- WHEN
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			status := fmt.Sprintf("%d", 200+(id%3))
			for i := 0; i < iters; i++ {
				metrics.RequestSent()
				metrics.RequestSucceeded(status)
				metrics.RequestFailed(status)
				metrics.RequestTookMillis(status, float64(i))
			}
		}(g)
	}
	wg.Wait()

	// ---- THEN — reached without panic; race detector covers data races when available
}

// Parallel use of one HttpServerLazyMetricsSet must be race-free (and must not panic).
func TestHttpServerLazyMetricsSet_ConcurrentUse_NoRace(t *testing.T) {
	// ---- GIVEN
	ensureMetricsInitialized(t)
	_ = kt_observability_monitoring.GetServerServeStartedCountTemplate()
	_ = kt_observability_monitoring.GetServerServeSucceededCountTemplate()
	_ = kt_observability_monitoring.GetServerServeFailedCountTemplate()
	_ = kt_observability_monitoring.GetServerServeProcessingTimeTemplate()

	metrics := kt_observability_monitoring.NewHttpServerLazyMetricsSet(
		"concurrent-server",
		kt_observability_monitoring.WithHttpServerId("s-race"),
	)
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut}
	const goroutines = 32
	const iters = 40

	// ---- WHEN
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func(id int) {
			defer wg.Done()
			req, err := http.NewRequest(methods[id%len(methods)], "http://example.local/x", nil)
			if err != nil {
				t.Errorf("NewRequest failed: %v", err)
				return
			}
			status := fmt.Sprintf("%d", 200+(id%3))
			for i := 0; i < iters; i++ {
				metrics.ServeStarted(req)
				metrics.ServeSucceeded(req, status)
				metrics.ServeFailed(req, status)
				metrics.ServeTookMillis(req, status, float64(i))
			}
		}(g)
	}
	wg.Wait()

	// ---- THEN — reached without panic; race detector covers data races when available
}

func ensureMetricsInitialized(t *testing.T) {
	t.Helper()
	// Init once for the whole package: templates register against MetricRegistry and must not be re-created on a new registry.
	initMetricsOnce.Do(func() {
		kt_observability_monitoring.InitMetrics()
		kt_observability_monitoring.SetGlobalLabels(map[string]any{
			"serviceName": "obs-test",
			"serviceVer":  "0.0.0",
			"host":        "test-host",
			"instId":      "test-inst",
		})
	})
	if kt_observability_monitoring.MetricRegistry == nil {
		t.Fatal("MetricRegistry is nil after InitMetrics")
	}
}

func findMetricFamily(t *testing.T, families []*dto.MetricFamily, name string) *dto.MetricFamily {
	t.Helper()
	for _, f := range families {
		if f.GetName() == name {
			return f
		}
	}
	t.Fatalf("metric family %q not found", name)
	return nil
}

func findMetricWithLabels(t *testing.T, family *dto.MetricFamily, wantLabels map[string]string) *dto.Metric {
	t.Helper()
	for _, m := range family.Metric {
		if metricHasLabels(m, wantLabels) {
			return m
		}
	}
	t.Fatalf("metric %q: no series matching labels %v", family.GetName(), wantLabels)
	return nil
}

func metricHasLabels(m *dto.Metric, wantLabels map[string]string) bool {
	present := make(map[string]string, len(m.Label))
	for _, l := range m.Label {
		present[l.GetName()] = l.GetValue()
	}
	for k, v := range wantLabels {
		if present[k] != v {
			return false
		}
	}
	return true
}

func assertCounterValue(t *testing.T, family *dto.MetricFamily, wantLabels map[string]string, want float64) {
	t.Helper()
	m := findMetricWithLabels(t, family, wantLabels)
	if m.Counter == nil {
		t.Fatalf("%s: expected counter", family.GetName())
	}
	if m.Counter.GetValue() != want {
		t.Errorf("%s: expected counter value %v, got %v", family.GetName(), want, m.Counter.GetValue())
	}
}

func assertSummarySampleCount(t *testing.T, family *dto.MetricFamily, wantLabels map[string]string, wantSamples uint64) {
	t.Helper()
	m := findMetricWithLabels(t, family, wantLabels)
	if m.Summary == nil {
		t.Fatalf("%s: expected summary", family.GetName())
	}
	if m.Summary.GetSampleCount() != wantSamples {
		t.Errorf("%s: expected sample_count %d, got %d", family.GetName(), wantSamples, m.Summary.GetSampleCount())
	}
}
