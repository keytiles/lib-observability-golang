package kt_observability_monitoring

import (
	"fmt"
	"reflect"
	"time"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-observability-golang/v2/pkg/kt_observability"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// A global, openly accessible MetricRegistry to register exposed metrics
	MetricRegistry *prometheus.Registry
	// The global key-value pairs used for each Metric - due to our Monitoring Standards
	globalMetricLabels prometheus.Labels

	// The global key-value pairs used for each Metric - due to our Monitoring Standards
	globalLabels map[string]any

	DefaultSummaryObjectives = map[float64]float64{
		0:    0.02,
		0.5:  0.02,
		0.95: 0.02,
		0.99: 0.02,
		1:    0.02,
	}

	// Unregistered sinks returned when metric instance creation soft-fails (label mismatch, nil vec, etc).
	discardedCounter  = prometheus.NewCounter(prometheus.CounterOpts{Name: "kt_obs_discarded_counter", Help: "soft-fail sink; not registered"})
	discardedGauge    = prometheus.NewGauge(prometheus.GaugeOpts{Name: "kt_obs_discarded_gauge", Help: "soft-fail sink; not registered"})
	discardedObserver = prometheus.NewSummary(prometheus.SummaryOpts{Name: "kt_obs_discarded_summary", Help: "soft-fail sink; not registered"})
)

// Builds a list of Prometheus metric labels from the given key-value map
func BuildMetricLabels(labels map[string]any) prometheus.Labels {

	metricLabels := prometheus.Labels{}

	for key, value := range labels {
		metricLabels[key] = fmt.Sprintf("%v", value)
	}

	return metricLabels
}

// Returns the first variable label name that also appears in global ConstLabels, or "".
func overlappingConstLabel(variableLabelNames []string) string {
	for _, name := range variableLabelNames {
		if _, ok := globalMetricLabels[name]; ok {
			return name
		}
	}
	return ""
}

// returns the current GlobalLabels - key-value pairs attached to all log events
func GetGlobalLabels() map[string]any {
	return globalLabels
}

// you can change the GlobalLabels with this - the key-value pairs attached to all log events
func SetGlobalLabels(labels map[string]any) {
	globalLabels = labels
	// transform immediately to Prometheus labels
	globalMetricLabels = BuildMetricLabels(labels)
}

// Initializing the Prometheus MetricRegistry. After this 'MetricRegistry' is available and global metric labels are set according to our Monitoring Standards.
// But feel free to change them via
// GetGlobalLabels() and SetGlobalLabels() methods!
func InitMetrics() {
	// let's create Metric registry
	MetricRegistry = prometheus.NewRegistry()
	// let's build up the global labels
	globalLabelsMap := kt_observability.BuildGlobalLabelsMap()
	SetGlobalLabels(globalLabelsMap)
}

// You get back a struct like this when you invoke GetSummaryMetricTemplate(), GetCounterMetricTemplate() or GetGaugeMetricTemplate() methods.
//
// Once you created the template you can register it into a MetricRegistry using .Register() method of it.
// After that you can use GetSummaryMetricInstance(), GetCounterMetricInstance() or GetGaugeMetricInstance() methods with corresponding parametrization
// to get back a concrete instance of your metric which is ready to be used to collect insights.
type MetricTemplate struct {
	fullyQualifiedName string
	customLabelNames   []string
	metricType         string

	isRegistered bool
	summaryVec   *prometheus.SummaryVec
	//summaryOpts  *prometheus.SummaryOpts
	counterVec *prometheus.CounterVec
	gaugeVec   *prometheus.GaugeVec

	_LOGGER *kt_logging.Logger
}

func (tpl *MetricTemplate) FullyQualifiedName() string {
	return tpl.fullyQualifiedName
}

func (tpl *MetricTemplate) CustomLabelNames() []string {
	return tpl.customLabelNames
}

func (tpl *MetricTemplate) MetricType() string {
	return tpl.metricType
}

func (tpl *MetricTemplate) IsRegistered() bool {
	return tpl.isRegistered
}

// Lazy-inits _LOGGER so zero-value templates can log without nil-deref.
func (tpl *MetricTemplate) ensureLogger() {
	if tpl._LOGGER == nil {
		tpl._LOGGER = kt_logging.GetLogger(PACKAGE_NAME + ".MetricTemplate")
	}
}

// True when reg is a nil interface or a nil concrete pointer/slice/etc. inside the interface.
func isNilRegisterer(reg prometheus.Registerer) bool {
	if reg == nil {
		return true
	}
	v := reflect.ValueOf(reg)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

// Use this method to register this template into a prometheus MetricRegistry.
// At this point you can use our global MetricRegistry (see global variable above!).
// After this you are ready to create concrete instances.
func (tpl *MetricTemplate) Register(reg prometheus.Registerer) {
	var err error
	// Bare nil interface must not call reflect.Value.IsNil (that panics on zero Value).
	if !isNilRegisterer(reg) {
		switch tpl.metricType {
		case "summary":
			if tpl.summaryVec == nil {
				err = fmt.Errorf("summary vec is nil - template was not created successfully")
			} else {
				err = reg.Register(tpl.summaryVec)
			}
		case "counter":
			if tpl.counterVec == nil {
				err = fmt.Errorf("counter vec is nil - template was not created successfully")
			} else {
				err = reg.Register(tpl.counterVec)
			}
		case "gauge":
			if tpl.gaugeVec == nil {
				err = fmt.Errorf("gauge vec is nil - template was not created successfully")
			} else {
				err = reg.Register(tpl.gaugeVec)
			}
		default:
			err = fmt.Errorf("unknown metric type: %v - don't know how to register", tpl.metricType)
		}
	} else {
		// oops it looks the registry was not initialized...
		err = fmt.Errorf("registry is Nil... was MetricRegistry initialized?")
	}
	if err != nil {
		tpl.ensureLogger()
		tpl._LOGGER.Warn("failed to register %v into registry - error: %v", tpl.ToString(), err)
	} else {
		tpl.isRegistered = true
	}
}

func (tpl *MetricTemplate) ToString() string {
	return fmt.Sprintf("MetricTemplate[metricType: %v, name: %v]", tpl.metricType, tpl.fullyQualifiedName)
}

// Creates a new Summary metric type template which is already using all GlobalMetricLabels plus you can pass in a set of
// customLabelNames by which filling them up with concrete values you will create your concrete metric instances.
// See: GetSummaryMetricInstance() method!
func GetSummaryMetricTemplate(opts prometheus.SummaryOpts, customLabelNames []string) MetricTemplate {
	opts.ConstLabels = globalMetricLabels
	opts.MaxAge = 60 * time.Second
	opts.AgeBuckets = 6
	opts.Objectives = DefaultSummaryObjectives

	customLabelNames = append(customLabelNames, "metricType")
	logger := kt_logging.GetLogger(PACKAGE_NAME + ".MetricTemplate")
	fqName := prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name)

	if clash := overlappingConstLabel(customLabelNames); clash != "" {
		logger.Warn("GetSummaryMetricTemplate(%v): global ConstLabel %q overlaps variable labels - soft-fail, vec not created", fqName, clash)
		return MetricTemplate{
			fullyQualifiedName: fqName,
			customLabelNames:   customLabelNames,
			metricType:         "summary",
			_LOGGER:            logger,
		}
	}

	return MetricTemplate{
		fullyQualifiedName: fqName,
		summaryVec:         prometheus.NewSummaryVec(opts, customLabelNames),
		customLabelNames:   customLabelNames,
		metricType:         "summary",
		_LOGGER:            logger,
	}
}

// Creates a concrete instance of a previously created Summary template by requiring you to provide concrete values
// for the customLabelNames you created the template with.
func GetSummaryMetricInstance(metricTemplate MetricTemplate, customLabels map[string]any) prometheus.Observer {
	if metricTemplate.metricType != "summary" {
		err := fmt.Sprintf(".GetSummaryMetricInstance() is invoked on %v but type of metric is different", metricTemplate.ToString())
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Error("ciritical error! app will panic - %v", err)
		panic(err)
	}
	if !metricTemplate.isRegistered {
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Warn("%v: metric instance creation was invoked but this template was not registered yet...", metricTemplate.ToString())
	}
	// Callers may pass nil; allocate so we can set metricType without panicking.
	if customLabels == nil {
		customLabels = make(map[string]any)
	}
	customLabels["metricType"] = metricTemplate.metricType

	if metricTemplate.summaryVec == nil {
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Warn("%v: summary vec is nil - returning discarded observer", metricTemplate.ToString())
		return discardedObserver
	}

	observerInstance, err := metricTemplate.summaryVec.GetMetricWith(BuildMetricLabels(customLabels))
	if err != nil {
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Warn("%v: failed to create summary instance (labels soft-fail) - error: %v", metricTemplate.ToString(), err)
		return discardedObserver
	}
	return observerInstance
}

func GetCounterMetricTemplate(opts prometheus.CounterOpts, customLabelNames []string) MetricTemplate {
	opts.ConstLabels = globalMetricLabels

	customLabelNames = append(customLabelNames, "metricType")
	logger := kt_logging.GetLogger(PACKAGE_NAME + ".MetricTemplate")
	fqName := prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name)

	if clash := overlappingConstLabel(customLabelNames); clash != "" {
		logger.Warn("GetCounterMetricTemplate(%v): global ConstLabel %q overlaps variable labels - soft-fail, vec not created", fqName, clash)
		return MetricTemplate{
			fullyQualifiedName: fqName,
			customLabelNames:   customLabelNames,
			metricType:         "counter",
			_LOGGER:            logger,
		}
	}

	return MetricTemplate{
		fullyQualifiedName: fqName,
		counterVec:         prometheus.NewCounterVec(opts, customLabelNames),
		customLabelNames:   customLabelNames,
		metricType:         "counter",
		_LOGGER:            logger,
	}
}

func GetCounterMetricInstance(metricTemplate MetricTemplate, customLabels map[string]any) prometheus.Counter {
	if metricTemplate.metricType != "counter" {
		err := fmt.Sprintf(".GetCounterMetricInstance() is invoked on %v but type of metric is different", metricTemplate.ToString())
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Error("ciritical error! app will panic - %v", err)
		panic(err)
	}
	if !metricTemplate.isRegistered {
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Warn("%v: metric instance creation was invoked but this template was not registered yet...", metricTemplate.ToString())
	}

	// Callers may pass nil; allocate so we can set metricType without panicking.
	if customLabels == nil {
		customLabels = make(map[string]any)
	}
	customLabels["metricType"] = metricTemplate.metricType
	if metricTemplate.counterVec == nil {
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Warn("%v: counter vec is nil - returning discarded counter", metricTemplate.ToString())
		return discardedCounter
	}
	counter, err := metricTemplate.counterVec.GetMetricWith(BuildMetricLabels(customLabels))
	if err != nil {
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Warn("%v: failed to create counter instance (labels soft-fail) - error: %v", metricTemplate.ToString(), err)
		return discardedCounter
	}
	return counter
}

func GetGaugeMetricTemplate(opts prometheus.GaugeOpts, customLabelNames []string) MetricTemplate {
	opts.ConstLabels = globalMetricLabels

	customLabelNames = append(customLabelNames, "metricType")
	logger := kt_logging.GetLogger(PACKAGE_NAME + ".MetricTemplate")
	fqName := prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name)

	if clash := overlappingConstLabel(customLabelNames); clash != "" {
		logger.Warn("GetGaugeMetricTemplate(%v): global ConstLabel %q overlaps variable labels - soft-fail, vec not created", fqName, clash)
		return MetricTemplate{
			fullyQualifiedName: fqName,
			customLabelNames:   customLabelNames,
			metricType:         "gauge",
			_LOGGER:            logger,
		}
	}

	return MetricTemplate{
		fullyQualifiedName: fqName,
		gaugeVec:           prometheus.NewGaugeVec(opts, customLabelNames),
		customLabelNames:   customLabelNames,
		metricType:         "gauge",
		_LOGGER:            logger,
	}
}

func GetGaugeMetricInstance(metricTemplate MetricTemplate, customLabels map[string]any) prometheus.Gauge {
	if metricTemplate.metricType != "gauge" {
		err := fmt.Sprintf(".GetGaugeMetricInstance() is invoked on %v but type of metric is different", metricTemplate.ToString())
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Error("ciritical error! app will panic - %v", err)
		panic(err)
	}
	if !metricTemplate.isRegistered {
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Warn("%v: metric instance creation was invoked but this template was not registered yet...", metricTemplate.ToString())
	}

	// Callers may pass nil; allocate so we can set metricType without panicking.
	if customLabels == nil {
		customLabels = make(map[string]any)
	}
	customLabels["metricType"] = metricTemplate.metricType
	if metricTemplate.gaugeVec == nil {
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Warn("%v: gauge vec is nil - returning discarded gauge", metricTemplate.ToString())
		return discardedGauge
	}
	gauge, err := metricTemplate.gaugeVec.GetMetricWith(BuildMetricLabels(customLabels))
	if err != nil {
		metricTemplate.ensureLogger()
		metricTemplate._LOGGER.Warn("%v: failed to create gauge instance (labels soft-fail) - error: %v", metricTemplate.ToString(), err)
		return discardedGauge
	}
	return gauge
}
