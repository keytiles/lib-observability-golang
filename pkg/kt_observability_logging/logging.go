package kt_observability_logging

import (
	"fmt"
	"reflect"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-observability-golang/v2/pkg/kt_observability"
)

// Builds a list of log labels from the given map
func BuildLogLabels(labels map[string]any) []kt_logging.Label {

	logLabels := make([]kt_logging.Label, 0, len(labels))

	for key, value := range labels {
		var label kt_logging.Label

		vt := reflect.TypeOf(value)
		if vt == nil {
			label = kt_logging.StringLabel(key, "<null>")
		} else {
			// Use reflect.Value so defined types (e.g. type MyInt int) convert without panicking.
			rv := reflect.ValueOf(value)
			switch rv.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				label = kt_logging.FloatLabel(key, float64(rv.Int()))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				label = kt_logging.FloatLabel(key, float64(rv.Uint()))
			case reflect.Float32, reflect.Float64:
				label = kt_logging.FloatLabel(key, rv.Float())
			case reflect.String:
				label = kt_logging.StringLabel(key, rv.String())
			case reflect.Bool:
				label = kt_logging.BoolLabel(key, rv.Bool())
			default:
				label = kt_logging.StringLabel(key, fmt.Sprintf("<'%v' value not supported>", rv.Kind()))
			}
		}
		logLabels = append(logLabels, label)
	}

	return logLabels
}

// builds the default key-value pairs due to our Logging Standards
func BuildDefaultGlobalLogLabels() []kt_logging.Label {

	globalLabelsMap := kt_observability.BuildGlobalLabelsMap()
	return BuildLogLabels(globalLabelsMap)

}
