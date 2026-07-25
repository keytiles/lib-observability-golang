# Plan: Review — docs, example app, and tests

- Created / last modified: 2026-07-25
- Target release folder: `development-plans/v2.0.1/`
- Status: implemented (first joint review/cleanup task on this library)

## Documentation references

Companion feature / architecture docs produced or updated by this plan:

- [docs/Architecture-v2.0.md](../../docs/Architecture-v2.0.md)
- [docs/LoggingObservability-v2.0.md](../../docs/LoggingObservability-v2.0.md)
- [docs/MetricsObservability-v2.0.md](../../docs/MetricsObservability-v2.0.md)
- [README.md](../../README.md) (light polish only)

Related agent rules followed:

- Docs rules (`agents/docs-rules-and-best-practices.md` → Keytiles docs + planning document standards)
- Coding rules (`agents/coding-rules-and-best-practices.md` → tests under `/tests`, mirroring `/pkg`)

## Why are we doing this?

- This was the first collaborative review of `lib-observability-golang` after exploring `pkg/`.
- Users (and future maintainers/agents) need a clear picture of what the library is and how Logging vs Metrics fit together.
- The existing “integration test” was not a test — it was a long-running manual demo app with no assertions. That naming was misleading.
- We want a clean split: runnable example for humans/docs, automated happy-path tests for CI, and versioned docs under `/docs`.

## What will be changed?

- Document library architecture and the two major features (Logging + Metrics) under `/docs`.
- Lightly polish `README.md` (keep structure; fix typos/paths; point to docs + example).
- Move `tests/integration_tests/` → `examples/simple-service/` (honest example app).
- Add real automated happy-path tests under `/tests` mirroring `/pkg`.
- Capture this work in a planning document (this file).

## Decisions we made

### Doc granularity

- Treat **Logging** and **Metrics** as the two feature docs (`LoggingObservability-v2.0`, `MetricsObservability-v2.0`).
- Add an **Architecture** doc for how packages fit together and shared global labels.
- Do **not** create a separate feature doc for `pkg/kt_observability` — it is a small shared foundation (cover in Architecture + brief mentions in feature docs).
- README stays the entry point; deep detail lives in `/docs` (no full README rewrite).

### Example vs tests

- Stop calling the demo an integration test; move it to `examples/simple-service/`.
- Keep the demo as a fire-up-and-play app (HTTP ping / fail, `/metrics`, simulated counters) for humans and for docs to reference.
- Add fast black-box tests under `tests/kt_observability_logging` and `tests/kt_observability_monitoring` (assert via `Gather()` / label APIs — do not automate the long-running process).
- Do not share packages between example and tests for now — both use the public API independently.

### Alternatives rejected

- Keep the app under `tests/` and only add asserts — still conflates “run forever” with CI.
- Example only, no tests — onboarding OK, regressions silent.
- Tests only, delete demo — loses a runnable walkthrough for docs/onboarding.

## Implementation steps

1. **Explore `pkg/` and agree documentation shape** — implemented  
   Confirmed packages, global labels, template → instance model, HTTP lazy metric sets; agreed Architecture + Logging + Metrics docs.

2. **Move demo to example app** — implemented  
   `tests/integration_tests/` → `examples/simple-service/` (including `http_handler`); removed old folder; small cleanup of duplicate `InitMetrics()` in the metrics exposer.

3. **Add automated happy-path tests** — implemented  
   - Logging: `BuildLogLabels` types / null / unsupported; default global keys  
   - Metrics: init + template instances + `Gather()` asserts; HTTP client/server lazy sets  

4. **Polish README** — implemented  
   Typos, fixed `metrics_templates.go` path, Documentation section, How to use → `examples/simple-service`.

5. **Write versioned docs** — implemented  
   `Architecture-v2.0.md`, `LoggingObservability-v2.0.md`, `MetricsObservability-v2.0.md` (AI-agent meta, what-changed, self-contained).

6. **Write this planning document** — implemented  
   Record requirements, decisions, and steps for the v2.0.1 review work.

## How to verify

- Run tests: `go test ./tests/...`
- Build / run example: `cd examples/simple-service && go run .`
  - HTTP: `http://localhost:8080/api/v1/ping` and `.../ping-fail`
  - Metrics: `http://localhost:9008/metrics`
- Skim README → Architecture → Logging/Metrics docs for cross-links and consistency

## Out of scope (for this plan)

- Broader API redesign of monitoring/logging packages
- CHANGELOG updates (unless requested separately)
- Godoc `Example*` functions
- Thin HTTP integration test that scrapes `/metrics` over the network (`Gather()` already covers the core contract)
