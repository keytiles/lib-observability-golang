# Plan: Eliminate runtime panics

- Created / last modified: 2026-07-26
- Target release folder: `development-plans/v2.0.1/`
- Status: analysis complete — Group 2.3 done; next 2.6; Group 1 last

## Documentation references

Companion docs this change will evolve (after decisions):

- [docs/Architecture-v2.0.md](../../docs/Architecture-v2.0.md)
- [docs/MetricsObservability-v2.0.md](../../docs/MetricsObservability-v2.0.md)
- [docs/LoggingObservability-v2.0.md](../../docs/LoggingObservability-v2.0.md)

Related agent rules:

- Docs / planning rules (`agents/docs-rules-and-best-practices.md`)
- Coding rules (`agents/coding-rules-and-best-practices.md`)

## Why are we doing this?

- A service that uses this library must **not** crash because the library raised a runtime panic.
- Observability code sits on hot paths (HTTP handlers, clients, metric recording); panics there take down the whole process.
- Today several paths intentionally `panic()`, and others can panic indirectly (Prometheus APIs, concurrent maps, type assertions).

## What will be changed?

- Inventory all panic / crash risks under `pkg/` — done (findings above).
- Eliminate **Group 2** unwanted crashes first, one finding per TDD increment.
- Eliminate **Group 1** explicit `panic()` last (likely API changes).
- Update tests, and docs/CHANGELOG as user-visible behavior settles.

## Findings (step 1 analysis)

Findings are split into two groups:

1. **Explicit `panic()`** — our code deliberately calls `panic(...)`
2. **Unwanted / indirect** — no `panic()` in our code, but runtime can still crash (stdlib, Prometheus, nil deref, races)

### Group 1 — Explicit `panic()` calls

These are intentional fail-fast checks written by us.

#### 1.1 Wrong metric template type in getters

- Files: [`monitoring.go`](../../pkg/kt_observability_monitoring/monitoring.go)
- Trigger: caller passes a Counter template into `GetSummaryMetricInstance` (or similar mismatch)
- Likelihood: low (API misuse)

Example (`GetSummaryMetricInstance`; Counter/Gauge getters are the same pattern):

```157:162:pkg/kt_observability_monitoring/monitoring.go
func GetSummaryMetricInstance(metricTemplate MetricTemplate, customLabels map[string]any) prometheus.Observer {
	if metricTemplate.metricType != "summary" {
		err := fmt.Sprintf(".GetSummaryMetricInstance() is invoked on %v but type of metric is different", metricTemplate.ToString())
		metricTemplate._LOGGER.Error("ciritical error! app will panic - %v", err)
		panic(err)
```

Same for Counter (~lines 191–195) and Gauge (~lines 219–223).

#### 1.2 Empty `of` when creating HTTP lazy metric sets

- Files: [`http_client_metrics.go`](../../pkg/kt_observability_monitoring/http_client_metrics.go), [`http_server_metrics.go`](../../pkg/kt_observability_monitoring/http_server_metrics.go)
- Trigger: `NewHttpClientLazyMetricsSet("")` or `NewHttpServerLazyMetricsSet("")`
- Likelihood: low–medium (bad config / empty string)

Client:

```30:33:pkg/kt_observability_monitoring/http_client_metrics.go
func NewHttpClientLazyMetricsSet(of string, opts ...HttpClientLazyMetricsSetOpt) *HttpClientLazyMetricsSet {
	if of == "" {
		panic("Can not create HttpClientLazyMetricsSet with empty 'of' parameter!")
	}
```

Server:

```30:33:pkg/kt_observability_monitoring/http_server_metrics.go
func NewHttpServerLazyMetricsSet(of string, opts ...HttpServerLazyMetricsSetOpt) *HttpServerLazyMetricsSet {
	if of == "" {
		panic("Can not create HttpServerLazyMetricsSet with empty 'of' parameter!")
	}
```

---

### Group 2 — Unwanted / indirect crash risks

No explicit `panic()` here, but a running service can still die.

#### 2.1 Concurrent map write races in HTTP lazy sets — **high**

- Files: [`http_client_metrics.go`](../../pkg/kt_observability_monitoring/http_client_metrics.go), [`http_server_metrics.go`](../../pkg/kt_observability_monitoring/http_server_metrics.go)
- Trigger: same metrics set used from multiple goroutines (normal HTTP usage); Go panics on concurrent map read/write
- Likelihood: **high** in real services

Example (client success path — maps updated without a mutex; server maps are the same idea):

```95:104:pkg/kt_observability_monitoring/http_client_metrics.go
func (m *HttpClientLazyMetricsSet) RequestSucceeded(withHttpStatusCode string) {
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
```

#### 2.2 Prometheus `.With()` label mismatch — **medium**

- File: [`monitoring.go`](../../pkg/kt_observability_monitoring/monitoring.go)
- Trigger: `customLabels` missing a required key, or has an unexpected extra key vs the template’s variable labels
- Likelihood: medium (easy caller mistake)

Example:

```201:202:pkg/kt_observability_monitoring/monitoring.go
	customLabels["metricType"] = metricTemplate.metricType
	return metricTemplate.counterVec.With(BuildMetricLabels(customLabels))
```

(Same pattern for Summary ~line 173 and Gauge ~line 230.)

#### 2.3 Nil `customLabels` map write — **medium** — **fixed (increment 2.3)**

- File: [`monitoring.go`](../../pkg/kt_observability_monitoring/monitoring.go)
- Trigger: `Get*MetricInstance(tpl, nil)` — assignment into a nil map panics
- Likelihood: medium if callers omit the map

Example (all three getters):

```166:166:pkg/kt_observability_monitoring/monitoring.go
	customLabels["metricType"] = metricTemplate.metricType
```

#### 2.4 `New*Vec` panics (const vs variable label clash / invalid names) — **medium**

- File: [`monitoring.go`](../../pkg/kt_observability_monitoring/monitoring.go) (and first use via [`metrics_templates.go`](../../pkg/kt_observability_monitoring/metrics_templates.go))
- Trigger: `SetGlobalLabels` includes a key that also appears as a variable label (`of`, `qualifier`, `metricType`, …), or invalid metric/label names — Prometheus constructor panics
- Likelihood: medium if services customize global labels carelessly

Example (const labels + variable names fed into Prometheus):

```177:184:pkg/kt_observability_monitoring/monitoring.go
func GetCounterMetricTemplate(opts prometheus.CounterOpts, customLabelNames []string) MetricTemplate {
	opts.ConstLabels = globalMetricLabels

	customLabelNames = append(customLabelNames, "metricType")

	return MetricTemplate{
		fullyQualifiedName: prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		counterVec:         prometheus.NewCounterVec(opts, customLabelNames),
```

#### 2.5 `Register` + `reflect.ValueOf(reg).IsNil()` — **low**

- File: [`monitoring.go`](../../pkg/kt_observability_monitoring/monitoring.go)
- Trigger: `Register(nil)` where `nil` is a **nil interface** (no concrete type) — `IsNil` on invalid Value panics
- Note: typed nil `*prometheus.Registry` in a `Registerer` interface is OK
- Also: register failure only Warns; `isRegistered` stays false — soft failure path, not always a panic

```104:108:pkg/kt_observability_monitoring/monitoring.go
func (tpl *MetricTemplate) Register(reg prometheus.Registerer) {
	var err error
	// if MetricRegistry was not initialized then the Registrer we get will point to a Nil instance - we have to detect that
	isNil := reflect.ValueOf(reg).IsNil()
	if !isNil {
```

#### 2.6 Logging type assertions on defined types — **low**

- File: [`logging.go`](../../pkg/kt_observability_logging/logging.go)
- Trigger: map value is a defined type with underlying int/string/bool (e.g. `type MyInt int`); `Kind` matches but `value.(int)` panics
- Likelihood: low

```25:26:pkg/kt_observability_logging/logging.go
			case reflect.Int:
				label = kt_logging.FloatLabel(key, float64(value.(int)))
```

#### 2.7 Nil `_LOGGER` on zero-value `MetricTemplate` — **low**

- File: [`monitoring.go`](../../pkg/kt_observability_monitoring/monitoring.go)
- Trigger: `MetricTemplate{}` (or otherwise nil `_LOGGER`) then hit Error/Warn paths → nil pointer dereference
- Can fire before / instead of the intentional type-mismatch panic in Group 1.1

```159:161:pkg/kt_observability_monitoring/monitoring.go
		err := fmt.Sprintf(".GetSummaryMetricInstance() is invoked on %v but type of metric is different", metricTemplate.ToString())
		metricTemplate._LOGGER.Error("ciritical error! app will panic - %v", err)
		panic(err)
```

---

### Notes (out of library `pkg/` scope)

- Example app `panic(err)` on HTTP listen failure in `examples/simple-service` — application concern.
- `InitMetrics` called twice / template singleton vs new registry — correctness/ops issue; may confuse failures but is not always a direct panic.

## Decisions we made

- Tackle **Group 2 (unwanted / indirect)** first — no intentional API redesign required for most of these; safer incremental fixes.
- Tackle **Group 1 (explicit `panic()`)** last — those checks likely need API changes (e.g. return `error` instead of `panic`), so they come after the soft-crash fixes.
- Work in **one-finding increments**, using **TDD**:
  1. Write a readable test for the **desired** behavior (“must not panic” / soft-fail / race-free).
  2. Confirm it is **red** (panic, nil deref, or `-race` failure).
  3. Fix the library code until the test is **green**.
  4. Only then move to the next finding.
- Prefer one scenario per test, GIVEN / WHEN / THEN comments (Keytiles test style).
- For **2.1 concurrent maps**: the red signal is primarily `go test -race` on a small parallel stress test (panic alone can be flaky); green means no panic and no race report.
- Exact soft-fail shape per finding (log + no-op vs return `error` vs validate-and-skip) is chosen inside each increment when writing the red test / fix — prefer the smallest change that removes the panic without a broad API break in Group 2.

## Implementation steps

### Done

1. **Analyze `pkg/` for panic / crash risks and document findings** — implemented.
2. **Increment — 2.3 Nil `customLabels` map write** — implemented  
   - Red: `Get*MetricInstance(tpl, nil)` must not panic.  
   - Fix: treat nil map as empty (allocate) before writing `metricType`.

### Group 2 — one increment each (TDD: red → fix → green)

Order: easiest / clearest tests first; concurrency last among Group 2.

3. **Increment — 2.6 Logging defined-type assertions** — planned  
   - Red: `BuildLogLabels` with a defined underlying type (e.g. `type MyInt int`) must not panic.  
   - Fix: convert via `reflect.Value` (or equivalent) instead of concrete type assert.

4. **Increment — 2.7 Nil `_LOGGER` on zero `MetricTemplate`** — planned  
   - Red: zero-value / nil-logger template paths must not nil-deref when logging.  
   - Fix: guard logger (lazy get / nil check) before Error/Warn.

5. **Increment — 2.2 Prometheus `.With()` label mismatch** — planned  
   - Red: incomplete or extra custom labels must not panic the process.  
   - Fix: validate labels against template (or use a non-panicking path) and soft-fail (log; return no-op / zero metric — decide in increment).

6. **Increment — 2.4 `New*Vec` const vs variable label clash** — planned  
   - Red: `SetGlobalLabels` overlapping variable names (e.g. `of`) then creating a template must not panic.  
   - Fix: detect overlap / invalid names before `New*Vec` and soft-fail (log; skip create / return error state — decide in increment).

7. **Increment — 2.5 `Register(nil)` + `reflect.IsNil`** — planned  
   - Red: `Register` with a nil interface must not panic.  
   - Fix: safe nil check before `reflect.Value.IsNil()`.

8. **Increment — 2.1 Concurrent map races in HTTP lazy sets** — planned  
   - Red: parallel use of the same metrics set fails under `go test -race` (and must not panic).  
   - Fix: synchronize map access (`sync.Mutex` or equivalent).

### Group 1 — last (likely API changes)

9. **Increment(s) — explicit `panic()` removal** — planned  
   - 1.1 Wrong template type in `Get*MetricInstance`  
   - 1.2 Empty `of` in `NewHttp*LazyMetricsSet`  
   - Design return/`error` (or other) API, TDD red tests for non-panicking behavior, then implement. May be one or more increments once API shape is agreed.

### Wrap-up

10. **Update companion docs + CHANGELOG** for the chosen soft-fail / API behavior — planned (after increments land, or per-increment if behavior is user-visible).  
11. **Final verify** — planned: `go test ./tests/...` and `go test -race` on monitoring HTTP lazy set tests.

## How to verify (per increment)

- New test is red before the fix; green after.
- Existing happy-path tests in `tests/kt_observability_*` stay green.
- For increment 2.1: `go test -race` on the monitoring test package.

