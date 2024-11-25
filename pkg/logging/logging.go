package kt_observability_logging

import (
	"fmt"
	"reflect"

	ktlogging "github.com/keytiles/lib-logging-golang"
	kt_observability "github.com/keytiles/lib-observability-golang"
)

// Builds a list of log labels from the given map
func BuildLogLabels(labels map[string]interface{}) []ktlogging.Label {

	logLabels := make([]ktlogging.Label, 0, len(labels))

	for key, value := range labels {
		var label ktlogging.Label

		vt := reflect.TypeOf(value)
		if vt == nil {
			label = ktlogging.StringLabel(key, "<null>")
		} else {
			valueParamType := vt.Kind()
			switch valueParamType {
			case reflect.Int:
				label = ktlogging.FloatLabel(key, float64(value.(int)))
			case reflect.Int8:
				label = ktlogging.FloatLabel(key, float64(value.(int8)))
			case reflect.Int16:
				label = ktlogging.FloatLabel(key, float64(value.(int16)))
			case reflect.Int32:
				label = ktlogging.FloatLabel(key, float64(value.(int32)))
			case reflect.Int64:
				label = ktlogging.FloatLabel(key, float64(value.(int64)))
			case reflect.Uint:
				label = ktlogging.FloatLabel(key, float64(value.(uint)))
			case reflect.Uint8:
				label = ktlogging.FloatLabel(key, float64(value.(uint8)))
			case reflect.Uint16:
				label = ktlogging.FloatLabel(key, float64(value.(uint16)))
			case reflect.Uint32:
				label = ktlogging.FloatLabel(key, float64(value.(uint32)))
			case reflect.Uint64:
				label = ktlogging.FloatLabel(key, float64(value.(uint64)))
			case reflect.Float32:
				label = ktlogging.FloatLabel(key, float64(value.(float32)))
			case reflect.Float64:
				label = ktlogging.FloatLabel(key, value.(float64))
			case reflect.String:
				label = ktlogging.StringLabel(key, value.(string))
			case reflect.Bool:
				label = ktlogging.BoolLabel(key, value.(bool))
			default:
				label = ktlogging.StringLabel(key, fmt.Sprintf("<'%v' value not supported>", valueParamType))
			}
		}
		logLabels = append(logLabels, label)
	}

	return logLabels
}

// builds the default key-value pairs due to our Logging Standards
func BuildDefaultGlobalLogLabels() []ktlogging.Label {

	globalLabelsMap := kt_observability.BuildGlobalLabelsMap()
	return BuildLogLabels(globalLabelsMap)

}
