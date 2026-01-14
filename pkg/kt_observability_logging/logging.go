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
			valueParamType := vt.Kind()
			switch valueParamType {
			case reflect.Int:
				label = kt_logging.FloatLabel(key, float64(value.(int)))
			case reflect.Int8:
				label = kt_logging.FloatLabel(key, float64(value.(int8)))
			case reflect.Int16:
				label = kt_logging.FloatLabel(key, float64(value.(int16)))
			case reflect.Int32:
				label = kt_logging.FloatLabel(key, float64(value.(int32)))
			case reflect.Int64:
				label = kt_logging.FloatLabel(key, float64(value.(int64)))
			case reflect.Uint:
				label = kt_logging.FloatLabel(key, float64(value.(uint)))
			case reflect.Uint8:
				label = kt_logging.FloatLabel(key, float64(value.(uint8)))
			case reflect.Uint16:
				label = kt_logging.FloatLabel(key, float64(value.(uint16)))
			case reflect.Uint32:
				label = kt_logging.FloatLabel(key, float64(value.(uint32)))
			case reflect.Uint64:
				label = kt_logging.FloatLabel(key, float64(value.(uint64)))
			case reflect.Float32:
				label = kt_logging.FloatLabel(key, float64(value.(float32)))
			case reflect.Float64:
				label = kt_logging.FloatLabel(key, value.(float64))
			case reflect.String:
				label = kt_logging.StringLabel(key, value.(string))
			case reflect.Bool:
				label = kt_logging.BoolLabel(key, value.(bool))
			default:
				label = kt_logging.StringLabel(key, fmt.Sprintf("<'%v' value not supported>", valueParamType))
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
