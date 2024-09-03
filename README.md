# lib-observability

Common code to add Metrics, Logs whatever - to Go services.

It's pulling in [lib-logging-golang](https://github.com/keytiles/lib-logging-golang) autmatically

# What does it bring?

## You can build Labels (for logs and metrics both)

Quickly according to our standards.

```go

import (
    // ...
	ktlogging "github.com/keytiles/lib-logging-golang"
	kt_observability "github.com/keytiles/lib-observability-golang"
	kt_observability_logging "github.com/keytiles/lib-observability-golang/logging"
	kt_observability_monitoring "github.com/keytiles/lib-observability-golang/monitoring"
    // ...
)

// this builds a set of labels comforms our standards
globalLabels := kt_observability.BuildGlobalLabelsMap()
// ... now optionaly you can add more globalLabels

// let's setup the global log labels
ktlogging.SetGlobalLabels(kt_observability_logging.BuildLogLabels(globalLabels))
// and now the metrics
kt_observability_monitoring.InitMetrics()
kt_observability_monitoring.SetGlobalLabels(globalLabels)
```

## You can build and handle simple Metrics

Our Metrics standards - supported by this lib out of the box - defines the following basic templates of metrics
 * execCount: Counter - to count executions
 * warningCount: Counter - to count warnings
 * errorCount: Counter - to count errors
 * processingTime: Summary - to measure processing time of something with default quantiles min(0), mean(0.5), max(1) and 0.95th and 0.99th percentiles

They look like this in Prometheus format

```
# HELP execCount Reports count executions of something (check 'of' attribute!)
# TYPE execCount counter
execCount{metricType="counter",of="mailFetch",qualifier="total"} 0

# HELP warningCount Reports count of a warning of something (check 'of' attribute!)
# TYPE warningCount counter
warningCount{metricType="counter",of="mailFetch",qualifier="total"} 0

# HELP errorCount Reports count of a failure of something (check 'of' attribute!)
# TYPE errorCount counter
errorCount{metricType="counter",of="mailFetch",qualifier="total"} 0
```

And this is how you can quickly create and use these from code:

```go

import (
    // ...
	kt_observability "github.com/keytiles/lib-observability-golang"
	kt_observability_monitoring "github.com/keytiles/lib-observability-golang/monitoring"
    // ...
)

var (
	// Metric instances - available globally
	mailFetchExecCountMetric          prometheus.Counter
	mailFetchFailedExecCountMetric    prometheus.Counter
)


// this builds a set of labels comforms our standards
globalLabels := kt_observability.BuildGlobalLabelsMap()
// ... now optionaly you can add more globalLabels

// and now the metrics
kt_observability_monitoring.InitMetrics()
kt_observability_monitoring.SetGlobalLabels(globalLabels)

// create some Counters!
mailFetchExecCountMetric = kt_observability_monitoring.GetCounterMetricInstance(metrics.GetExecCountTemplate(), map[string]interface{}{"of": "mailFetch"})
mailFetchFailedExecCountMetric = kt_observability_monitoring.GetCounterMetricInstance(metrics.GetErrorCountTemplate(), map[string]interface{}{"of": "mailFetch"})

// and now let's bump
function fetchMails() {
    mailFetchExecCountMetric.Inc()

    // ... do it

    if error != nil {
        // oops we failed!
        mailFetchFailedExecCountMetric.Inc()
    }
}

```


