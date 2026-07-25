package kt_observability_monitoring_test

import (
	"net/http"
	"sync"
	"testing"

	"github.com/keytiles/lib-observability-golang/v2/pkg/kt_observability_monitoring"
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
