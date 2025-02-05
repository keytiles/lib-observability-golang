
# release 1.5.1

Fixes:
 * Observability: renaming method NewHttpClientLazyMetrics -> NewHttpClientLazyMetricsSet (not breaking change as feature is totally new)

# release 1.5.0

New features:
 * Observability: added HttpClientLazyMetricsSet for easier and full standard Metrics for HTTP clients

# release 1.4.1

Fixes:
 * Observability: forgot about request success counter in sync client metrics


# release 1.4.0

New features:
 * Observability: added some useful generic metrics templates you can use in any synchronous clients - e.g. http or gRPC clients.

# release 1.3.0

New features:
 * restructuring internally
 * providing a test app to test the functionality (/tests/integratio_tests)

Bug fixes:
 * fixing some typos and stupid method names

Breaking changes:


# release 1.2.1

New features:

Bug fixes:
 * removed 'custom' label from built in metric templates as not needed - and also updated README to show correct instantiation

Breaking changes:


# release 1.2.0

New features:
 * Added support to handle simple way our Metric standards - introducing
    * kt_observability_monitoring.GetExecCountTemplate()
    * kt_observability_monitoring.GetErrorCountTemplate()
    * kt_observability_monitoring.GetWarningCountTemplate()
    * kt_observability_monitoring.ProcessingTimeTemplate()
 * From now on kt_observability_monitoring also has GetGlobalLabels() and SetGlobalLabels() methods - just like it work in kt_logging

Bug fixes:

Breaking changes:


# release 1.1.0

New features:
 * Added kt_observability_logging.logging.go helper - this brings BuildDefaultGlobalLogLabels() method so log labels can setup similar to Metric labels

Bug fixes:

Breaking changes:


# release 1.0.2

Initial release

# release 1.0.1

Tried to fix issues but failed - retracted

# release 1.0.0

Original release but wrong repo name - retracted