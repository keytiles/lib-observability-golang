# Plan: Eliminate runtime panics

- Created / last modified: 2026-07-26
- Target release folder: `development-plans/v2.1.0/`
- Status: Group 2 + Group 1 complete (OrFault + empty `of`) — docs/CHANGELOG wrap-up done for Metrics v2.1; final verify as needed

## Documentation references

Companion docs this change will evolve (after decisions):

- [docs/Architecture-v2.0.md](../../docs/Architecture-v2.0.md)
- [docs/MetricsObservability-v2.1.md](../../docs/MetricsObservability-v2.1.md) (target; v2.0 remains the prior baseline)
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
- Eliminate **Group 1** explicit `panic()` last — getters via `*OrFault` + deprecated soft-fail wrappers; then empty `of`.
- Update tests, and docs/CHANGELOG as user-visible behavior settles.

## Findings (step 1 analysis)

Findings are split into two groups:

1. **Explicit `panic()`** — our code deliberately calls `panic(...)`
2. **Unwanted / indirect** — no `panic()` in our code, but runtime can still crash (stdlib, Prometheus, nil deref, races)

### Group 1 — Explicit `panic()` calls

These are intentional fail-fast checks written by us.

#### 1.1 Wrong metric template type in getters — **fixed (increments 9a–9c)**

- Files: [`monitoring.go`](../../pkg/kt_observability_monitoring/monitoring.go)
- Trigger: caller passes a Counter template into `GetSummaryMetricInstance` (or similar mismatch)
- Likelihood: low (API misuse)
- **Decision (locked 2026-07-26):** non-breaking deprecate + `OrFault` preferred API using [`kt_errors.Fault`](https://github.com/keytiles/lib-errorhandling-golang).

Preferred:

```go
func GetCounterMetricInstanceOrFault(...) (prometheus.Counter, kt_errors.Fault)
func GetSummaryMetricInstanceOrFault(...) (prometheus.Observer, kt_errors.Fault)
func GetGaugeMetricInstanceOrFault(...) (prometheus.Gauge, kt_errors.Fault)
```

Deprecated wrappers call `OrFault`; on Fault → Warn + discarded metric (no panic).

#### 1.2 Empty `of` when creating HTTP lazy metric sets — **fixed (increment 9d)**

- Files: [`http_client_metrics.go`](../../pkg/kt_observability_monitoring/http_client_metrics.go), [`http_server_metrics.go`](../../pkg/kt_observability_monitoring/http_server_metrics.go)
- Trigger: `NewHttpClientLazyMetricsSet("")` or `NewHttpServerLazyMetricsSet("")`
- Likelihood: low–medium (bad config / empty string)
- **Decision:** same `OrFault` + deprecate pattern as 1.1. Empty `of` → `ValidationFault` + `VALIDATION_ERRCODE_MISSING_MANDATORY`. Deprecated constructors Warn and create with placeholder `of="-"` (matches other label defaults); no panic.

Preferred:

```go
func NewHttpClientLazyMetricsSetOrFault(of string, opts ...) (*HttpClientLazyMetricsSet, kt_errors.Fault)
func NewHttpServerLazyMetricsSetOrFault(of string, opts ...) (*HttpServerLazyMetricsSet, kt_errors.Fault)
```

Client:

```go
func NewHttpClientLazyMetricsSet(of string, opts ...HttpClientLazyMetricsSetOpt) *HttpClientLazyMetricsSet {
	// wraps OrFault; empty of → Warn + placeholder "-"
}
```

Server: same pattern.
---

### Group 2 — Unwanted / indirect crash risks

No explicit `panic()` here, but a running service can still die.

#### 2.1 Concurrent map write races in HTTP lazy sets — **high** — **fixed (increment 2.1)**

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

#### 2.2 Prometheus `.With()` label mismatch — **medium** — **fixed (increment 2.2)**

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

#### 2.4 `New*Vec` panics (const vs variable label clash / invalid names) — **medium** — **fixed (increment 2.4)**

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

#### 2.5 `Register` + `reflect.ValueOf(reg).IsNil()` — **low** — **fixed (increment 2.5)**

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

#### 2.6 Logging type assertions on defined types — **low** — **fixed (increment 2.6)**

- File: [`logging.go`](../../pkg/kt_observability_logging/logging.go)
- Trigger: map value is a defined type with underlying int/string/bool (e.g. `type MyInt int`); `Kind` matches but `value.(int)` panics
- Likelihood: low

```25:26:pkg/kt_observability_logging/logging.go
			case reflect.Int:
				label = kt_logging.FloatLabel(key, float64(value.(int)))
```

#### 2.7 Nil `_LOGGER` on zero-value `MetricTemplate` — **low** — **fixed (increment 2.7)**

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
- Tackle **Group 1 (explicit `panic()`)** last — after soft-crash fixes; API shape agreed for getters (below).
- Work in **one-finding increments**, using **TDD**:
  1. Write a readable test for the **desired** behavior (“must not panic” / soft-fail / race-free).
  2. Confirm it is **red** (panic, nil deref, or `-race` failure).
  3. Fix the library code until the test is **green**.
  4. Only then move to the next finding.
- Prefer one scenario per test, GIVEN / WHEN / THEN comments (Keytiles test style).
- For **2.1 concurrent maps**: the red signal is primarily `go test -race` on a small parallel stress test (panic alone can be flaky); green means no panic and no race report.
- Exact soft-fail shape per finding (log + no-op vs return `error` vs validate-and-skip) is chosen inside each increment when writing the red test / fix — prefer the smallest change that removes the panic without a broad API break in Group 2.

### Group 1.1 getter API (locked)

- **Preferred:** `GetCounterMetricInstanceOrFault(...) (prometheus.Counter, kt_errors.Fault)` — same `GetCounter…` prefix for IDE autocomplete; returns bare `nil` Fault on success, `(nil, fault)` on failure.
- **Deprecated (kept):** `GetCounterMetricInstance(...) prometheus.Counter` — wraps `OrFault`; on any Fault → **Warn** + `discardedCounter` (no panic). Non-breaking for existing call sites.
- **Fault library:** [lib-errorhandling-golang](https://github.com/keytiles/lib-errorhandling-golang) (`kt_errors.Fault`). No circular dependency (errorhandling does not import observability). Add `github.com/keytiles/lib-errorhandling-golang/v2` to `go.mod`.
- **`OrFault` failure modes** (all return Fault; deprecated path swallows → discarded):
  - Wrong template type → `ValidationFault` + `VALIDATION_ERRCODE_WRONG_DATATYPE`; `WithSource(PACKAGE_NAME, …)`
  - Nil `counterVec` → `IllegalStateFault`
  - Label mismatch from `GetMetricWith` → `ValidationFault` (cause wrapped when practical)
- **Not a Fault:** nil `customLabels` → empty map; unregistered template → Warn only, then continue.
- **Order:** Counter / Summary / Gauge `OrFault` first; then 1.2 empty `of` with the same constructor `*OrFault` + deprecate pattern.

### Group 1.2 HTTP lazy-set constructors (locked)

- **Preferred:** `NewHttpClientLazyMetricsSetOrFault` / `NewHttpServerLazyMetricsSetOrFault` — empty `of` → `(nil, ValidationFault` + `MISSING_MANDATORY)`.
- **Deprecated:** existing `NewHttp*LazyMetricsSet` — on Fault → Warn + create with placeholder `of="-"` so the set stays usable (no nil deref on later calls).

## Implementation steps

### Done

1. **Analyze `pkg/` for panic / crash risks and document findings** — implemented.

#### Group 2 — unwanted / indirect (implemented)

No intentional API redesign; soft-fail / harden so a running service does not die on these paths. Order was easiest tests first; concurrency last.

2. **Increment — 2.3 Nil `customLabels` map write** — implemented  
   - Red: `Get*MetricInstance(tpl, nil)` must not panic.  
   - Fix: treat nil map as empty (allocate) before writing `metricType`.
3. **Increment — 2.6 Logging defined-type assertions** — implemented  
   - Red: `BuildLogLabels` with defined underlying types must not panic.  
   - Fix: convert via `reflect.Value` (`Int`/`Uint`/`Float`/`String`/`Bool`).
4. **Increment — 2.7 Nil `_LOGGER` on zero `MetricTemplate`** — implemented  
   - Red: zero-value `Register` must not nil-deref.  
   - Fix: `ensureLogger()` before Error/Warn.
5. **Increment — 2.2 Prometheus `.With()` label mismatch** — implemented  
   - Red: missing/extra custom labels must not panic.  
   - Fix: `GetMetricWith` + Warn + package-level discarded Counter/Gauge/Observer.
6. **Increment — 2.4 Const vs variable label clash** — implemented  
   - Red: overlapping global ConstLabel (e.g. `of`) must not panic template create/use.  
   - Fix: pre-check overlap before `New*Vec` (nil vec + Warn); nil-vec-safe `Register`/getters.  
   - Note: current Prometheus `New*Vec` often does not panic on overlap; Register previously soft-failed — pre-check still applied for clear soft-fail.
7. **Increment — 2.5 `Register(nil)` + `reflect.IsNil`** — implemented  
   - Red: nil Registerer interface must not panic.  
   - Fix: `isNilRegisterer` checks bare nil before `IsNil`.
8. **Increment — 2.1 Concurrent map races in HTTP lazy sets** — implemented  
   - Red: parallel stress panics with `concurrent map read and map write` (race detector unavailable here without gcc).  
   - Fix: `sync.Mutex` on client/server lazy sets; also `sync.Once` for metric template singleton init (needed for concurrent first-use safety).

### Group 1 — explicit `panic()` (API agreed for getters)

9a. **Increment — 1.1 Counter `OrFault`** — implemented  
   - Dep `lib-errorhandling-golang/v2`; `GetCounterMetricInstanceOrFault` owns logic.  
   - Wrong type → `ValidationFault` + `VALIDATION_ERRCODE_WRONG_DATATYPE`; nil vec → `IllegalStateFault`; label mismatch → `ValidationFault` + cause.  
   - Deprecated `GetCounterMetricInstance` wraps + Warn + `discardedCounter`.
9b. **Increment — 1.1 Summary `OrFault`** — implemented  
   - Same pattern: `GetSummaryMetricInstanceOrFault` + deprecate `GetSummaryMetricInstance`.
9c. **Increment — 1.1 Gauge `OrFault`** — implemented  
   - Same pattern: `GetGaugeMetricInstanceOrFault` + deprecate `GetGaugeMetricInstance`.
9d. **Increment — 1.2 Empty `of` in `NewHttp*LazyMetricsSet`** — implemented  
   - `NewHttpClientLazyMetricsSetOrFault` / `NewHttpServerLazyMetricsSetOrFault`; empty `of` → `ValidationFault` + `MISSING_MANDATORY`.  
   - Deprecated constructors wrap + Warn + placeholder `of="-"`.

### Wrap-up

10. **Update companion docs + CHANGELOG** for the chosen soft-fail / API behavior — implemented  
    - [MetricsObservability-v2.1.md](../../docs/MetricsObservability-v2.1.md) crafted (self-contained; OrFault + deprecate soft-fail).  
    - CHANGELOG `2.1.0` notes updated; README / Architecture / Logging related links pointed at Metrics v2.1 / plan folder `v2.1.0`.  
11. **Final verify** — partial: `go test` on logging+monitoring packages green (Group 1.1 + 1.2); `go test -race` skipped on this Windows env (no CGO/gcc). Re-run `-race` where gcc is available.

## How to verify (per increment)

- New test is red before the fix; green after.
- Existing happy-path tests in `tests/kt_observability_*` stay green.
- For increment 2.1: `go test -race` on the monitoring test package.

