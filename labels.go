package kt_observability

import (
	"os"
)

// Builds the default key-value pairs due to our Logging / Monitoring standards (TBD!!!)
func BuildGlobalLabelsMap() map[string]interface{} {
	globalLabels := make(map[string]interface{})

	appName := os.Getenv("CONTAINER_NAME")
	if appName == "" {
		appName = "?"
	}
	globalLabels["appName"] = appName

	appVer := os.Getenv("CONTAINER_VERSION")
	if appVer == "" {
		appVer = "?"
	}
	globalLabels["appVer"] = appVer

	host := os.Getenv("HOSTNAME")
	if host == "" {
		host = "?"
	}
	globalLabels["host"] = host

	instId := os.Getenv("INSTANCE_ID")
	if instId == "" {
		instId = "?"
	}
	globalLabels["instId"] = instId

	return globalLabels
}
