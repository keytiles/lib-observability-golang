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
)

// Builds a list of Prometheus metric labels from the given key-value map
func BuildMetricLabels(labels map[string]any) prometheus.Labels {

	metricLabels := prometheus.Labels{}

	for key, value := range labels {
		metricLabels[key] = fmt.Sprintf("%v", value)
	}

	return metricLabels
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

// Use this method to register this template into a prometheus MetricRegistry.
// At this point you can use our global MetricRegistry (see global variable above!).
// After this you are ready to create concrete instances.
func (tpl *MetricTemplate) Register(reg prometheus.Registerer) {
	var err error
	// if MetricRegistry was not initialized then the Registrer we get will point to a Nil instance - we have to detect that
	isNil := reflect.ValueOf(reg).IsNil()
	if !isNil {
		switch tpl.metricType {
		case "summary":
			err = reg.Register(tpl.summaryVec)
		case "counter":
			err = reg.Register(tpl.counterVec)
		case "gauge":
			err = reg.Register(tpl.gaugeVec)
		default:
			err = fmt.Errorf("unknown metric type: %v - don't know how to register", tpl.metricType)
		}
	} else {
		// oops it looks the registry was not initialized...
		err = fmt.Errorf("registry is Nil... was MetricRegistry initialized?")
	}
	if err != nil {
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

	return MetricTemplate{
		fullyQualifiedName: prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		summaryVec:         prometheus.NewSummaryVec(opts, customLabelNames),
		//summaryOpts:        &opts,
		customLabelNames: customLabelNames,
		metricType:       "summary",
		_LOGGER:          kt_logging.GetLogger("keytiles.observability.monitoring.MetricTemplate"),
	}
}

// Creates a concrete instance of a previously created Summary template by requiring you to provide concrete values
// for the customLabelNames you created the template with.
func GetSummaryMetricInstance(metricTemplate MetricTemplate, customLabels map[string]any) prometheus.Observer {
	if metricTemplate.metricType != "summary" {
		err := fmt.Sprintf(".GetSummaryMetricInstance() is invoked on %v but type of metric is different", metricTemplate.ToString())
		metricTemplate._LOGGER.Error("ciritical error! app will panic - %v", err)
		panic(err)
	}
	if !metricTemplate.isRegistered {
		metricTemplate._LOGGER.Warn("%v: metric instance creation was invoked but this template was not registered yet...", metricTemplate.ToString())
	}
	customLabels["metricType"] = metricTemplate.metricType

	// this is not working for some reason
	// summaryInstance := prometheus.NewSummary(*metricTemplate.summaryOpts)
	// MetricRegistry.Register(summaryInstance)
	// return summaryInstance

	observerInstance := metricTemplate.summaryVec.With(BuildMetricLabels(customLabels))
	return observerInstance
}

func GetCounterMetricTemplate(opts prometheus.CounterOpts, customLabelNames []string) MetricTemplate {
	opts.ConstLabels = globalMetricLabels

	customLabelNames = append(customLabelNames, "metricType")

	return MetricTemplate{
		fullyQualifiedName: prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		counterVec:         prometheus.NewCounterVec(opts, customLabelNames),
		customLabelNames:   customLabelNames,
		metricType:         "counter",
		_LOGGER:            kt_logging.GetLogger("keytiles.observability.monitoring.MetricTemplate"),
	}
}

func GetCounterMetricInstance(metricTemplate MetricTemplate, customLabels map[string]any) prometheus.Counter {
	if metricTemplate.metricType != "counter" {
		err := fmt.Sprintf(".GetCounterMetricInstance() is invoked on %v but type of metric is different", metricTemplate.ToString())
		metricTemplate._LOGGER.Error("ciritical error! app will panic - %v", err)
		panic(err)
	}
	if !metricTemplate.isRegistered {
		metricTemplate._LOGGER.Warn("%v: metric instance creation was invoked but this template was not registered yet...", metricTemplate.ToString())
	}

	customLabels["metricType"] = metricTemplate.metricType
	return metricTemplate.counterVec.With(BuildMetricLabels(customLabels))
}

func GetGaugeMetricTemplate(opts prometheus.GaugeOpts, customLabelNames []string) MetricTemplate {
	opts.ConstLabels = globalMetricLabels

	customLabelNames = append(customLabelNames, "metricType")

	return MetricTemplate{
		fullyQualifiedName: prometheus.BuildFQName(opts.Namespace, opts.Subsystem, opts.Name),
		gaugeVec:           prometheus.NewGaugeVec(opts, customLabelNames),
		customLabelNames:   customLabelNames,
		metricType:         "gauge",
		_LOGGER:            kt_logging.GetLogger("keytiles.observability.monitoring.MetricTemplate"),
	}
}

func GetGaugeMetricInstance(metricTemplate MetricTemplate, customLabels map[string]any) prometheus.Gauge {
	if metricTemplate.metricType != "gauge" {
		err := fmt.Sprintf(".GetGaugeMetricInstance() is invoked on %v but type of metric is different", metricTemplate.ToString())
		metricTemplate._LOGGER.Error("ciritical error! app will panic - %v", err)
		panic(err)
	}
	if !metricTemplate.isRegistered {
		metricTemplate._LOGGER.Warn("%v: metric instance creation was invoked but this template was not registered yet...", metricTemplate.ToString())
	}

	customLabels["metricType"] = metricTemplate.metricType
	return metricTemplate.gaugeVec.With(BuildMetricLabels(customLabels))
}
