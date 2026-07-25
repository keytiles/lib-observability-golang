# lib-observability

Common code to add Metrics, Logs whatever - to Go services.

It's pulling in [lib-logging-golang](https://github.com/keytiles/lib-logging-golang) automatically. And also using the [prometheus library](https://github.com/prometheus/client_golang) to spin up the Prometheus http metrics endpoint.

# Documentation

More detail lives under [`docs/`](docs/):

- [Architecture-v2.0.md](docs/Architecture-v2.0.md) — how the library fits together
- [LoggingObservability-v2.0.md](docs/LoggingObservability-v2.0.md) — logging labels helpers
- [MetricsObservability-v2.0.md](docs/MetricsObservability-v2.0.md) — Prometheus metrics templates and HTTP helpers

# What does it bring?

## Standards!

Most of all. Crossing over both logging and monitoring.

### Labels

Labels are a very important part of both. They are the key-value pairs which are decorating log events and also metric instances. And central systems provide the way for filtering/grouping based on labels.

When you are writing a service there are certain labels which makes sense to be present in all log events and all metric instances. Therefore these can be considered as **global labels**. For example "service name" or "host" or "service version". With the lib - as you will see below in the example - you can simply build these and then just register them into both: logs and metrics.

### Metrics standards

You create and expose Metrics. Cool! But this is something which in itself does not provide any value. You also need to collect and store them (Prometheus, VictoriaMetrics etc) and create dashboards / alerting out of them (Grafana).

This is not complicated. Technically speaking. However doing it right and (centrally) maintainable way it is more challenging. Experience shows that on the long term too much freedom on Service developer side quickly blowing up the maintainability e.g. of Grafana dashboards. So the question is: can we do it better?

Certainly yes! But to achieve a healthy balance between Service developer freedom AND Grafana dashboards you need some standards.

Therefore the library comes with a concept distinguishing
 * Metrics templates and
 * Metric instances

Every Metric instance you create should be derived from one of the Metric templates! What is a template? Well not much. It has
 * A metric type, e.g. Counter or Summary
 * The fixed name of this metric, e.g. "execCount"
 * A (pre)fixed set of labels. When one creates a concrete instance of this template he 
    * must provide value for ALL of those pre-defined labels (empty value is OK)
    * can not add more labels

The library pre-defines a few Metrics Templates (which are typically enough in any application - you find them in [metrics_templates.go](pkg/kt_observability_monitoring/metrics_templates.go)), these are:
 * ExecCount - a Counter "of" something ("of" is a label)
 * ErrorCount - a Counter "of" something ("of" is a label) which represents a failure/error. Normally you would like to see 0 here right? And build alerting around these.
 * WarningCount - a Counter "of" something ("of" is a label) which represents a warning. More relaxed compared to errors but still can be important to keep an eye on.
 * ProcessingTime - a Summary "of" something ("of" is a label) with which you can measure time of some processing.

Once the template is created it is easy to create concrete instances of that template. But all the instances you create will 100% sure conform the "standards" the template defined.


# How to use

Just take a quick look into the attached example application!

See [examples/simple-service](examples/simple-service/) !

You can even run it with

```
cd examples/simple-service
go run .
```

Then hit `http://localhost:8080/api/v1/ping` (or `ping-fail`) and scrape metrics at `http://localhost:9008/metrics`.
