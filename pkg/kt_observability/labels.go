package kt_observability

import (
	"os"
)

// Builds the default key-value pairs due to our Logging / Monitoring standards (TBD!!!)
func BuildGlobalLabelsMap() map[string]any {
	globalLabels := make(map[string]any)

	serviceName := os.Getenv("SERVICE_NAME")
	if serviceName == "" {
		serviceName = os.Getenv("CONTAINER_NAME")
	}
	if serviceName == "" {
		serviceName = "?"
	}
	globalLabels["serviceName"] = serviceName

	serviceVer := os.Getenv("SERVICE_VERSION")
	if serviceVer == "" {
		serviceVer = os.Getenv("CONTAINER_VERSION")
	}
	if serviceVer == "" {
		serviceVer = "?"
	}
	globalLabels["serviceVer"] = serviceVer

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
