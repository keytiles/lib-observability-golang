package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-observability-golang/v2/pkg/kt_observability_logging"
	"github.com/keytiles/lib-observability-golang/v2/pkg/kt_observability_monitoring"
	http_handler "github.com/keytiles/lib-observability-golang/v2/tests/integration_tests/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	brokerTopic1_messageArrived   prometheus.Counter
	brokerTopic1_messageRetried   prometheus.Counter
	brokerTopic1_processingFailed prometheus.Counter
	brokerTopic1_processingTime   prometheus.Observer

	threadExecCount int
)

func main() {

	// build some global labels!
	globalLabels := make(map[string]any)
	globalLabels["globalLabel1"] = "value1"
	globalLabels["globalLabel2"] = 5

	// set the global labels for logging
	kt_logging.SetGlobalLabels(kt_observability_logging.BuildLogLabels(globalLabels))

	// init metrics and set global labels for metrics too
	kt_observability_monitoring.InitMetrics()
	kt_observability_monitoring.SetGlobalLabels(globalLabels)

	LOG := kt_logging.GetLogger("main")

	LOG.Info("starting up application...")

	// let's establish the prometheus http endpoint
	exposeMetrics(LOG, globalLabels)

	// create simple HTTP server
	httpHost := "0.0.0.0"
	httpPort := 8080
	webHandler := http_handler.NewHttpServerHandler()
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/ping", webHandler.ServePingOK).Methods("GET")
	router.HandleFunc("/api/v1/ping-fail", webHandler.ServePingFailed).Methods("GET")
	LOG.Info("starting http server on host: %s, port: %d.", httpHost, httpPort)
	go func() {
		err := http.ListenAndServe(fmt.Sprintf("%s:%d", httpHost, httpPort), router)
		if err != nil {
			panic(err)
		}
	}()
	LOG.Info("http server is up! You can execute now")
	LOG.Info("    http://%s:%d/api/v1/ping - for successful request", httpHost, httpPort)
	LOG.Info("    http://%s:%d/api/v1/ping-fail - for failed request", httpHost, httpPort)

	// now create some metrics instances -  start with exec/error/warning Counters
	brokerTopic1_messageArrived = kt_observability_monitoring.GetCounterMetricInstance(
		kt_observability_monitoring.GetExecCountTemplate(),
		map[string]any{"of": "msgArrived", "qualifier": "broker-topic-1"},
	)
	brokerTopic1_processingFailed = kt_observability_monitoring.GetCounterMetricInstance(
		kt_observability_monitoring.GetErrorCountTemplate(),
		map[string]any{"of": "msgProcessingFailed", "qualifier": "broker-topic-1"},
	)
	brokerTopic1_messageRetried = kt_observability_monitoring.GetCounterMetricInstance(
		kt_observability_monitoring.GetWarningCountTemplate(),
		map[string]any{"of": "msgProcessingRetried", "qualifier": "broker-topic-1"},
	)
	// and now some processing time simulation
	brokerTopic1_processingTime = kt_observability_monitoring.GetSummaryMetricInstance(
		kt_observability_monitoring.GetProcessingTimeTemplate(),
		map[string]any{"of": "msgProcessingRetried", "qualifier": "broker-topic-1"},
	)

	LOG.Info("starting main thread...")

	ctx, stopAndExitFunc := context.WithCancel(context.Background())
	simulateMetricsTicker := time.NewTicker(time.Second * 1)

	go func() {
		LOG := kt_logging.GetLogger("main.thread")

		doRun := true
		for doRun {
			select {
			// first let's check if we received the stop signal
			case <-ctx.Done():
				// lets break out from the loop and finish go routine
				doRun = false
			case <-simulateMetricsTicker.C:
				simulateAppLogic()
			}
		}

		LOG.Info("exited from for loop - completed")
	}()

	LOG.Info("app started!")

	waitUntilSigtermArrives()

	simulateMetricsTicker.Stop()

	// we cancel the context -> both threads will get the signal
	stopAndExitFunc()

	LOG.Info("app stopped, exiting...")

}

func exposeMetrics(LOG *kt_logging.Logger, globalLabels map[string]any) {

	kt_observability_monitoring.InitMetrics()
	kt_observability_monitoring.SetGlobalLabels(globalLabels)

	// Expose prometheus metrics via http at localhost:9008/metrics
	port := "9008"
	path := "/metrics"

	prometheusExposer := http.NewServeMux()
	prometheusExposer.Handle(
		path,
		promhttp.HandlerFor(kt_observability_monitoring.MetricRegistry, promhttp.HandlerOpts{Registry: kt_observability_monitoring.MetricRegistry}),
	)

	go http.ListenAndServe(":"+port, prometheusExposer)

	LOG.Info("Prometheus exporter listening at http://localhost:%s%s", port, path)
}

// This method blocks the execution until process is not stopped
func waitUntilSigtermArrives() {
	LOG := kt_logging.GetLogger("main")

	// let's wait now the exit signal
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)
	LOG.Info("Now waiting for kill signal...")
	<-done // Will block here until user hits ctrl+c

	LOG.Info("kill signal arrived - exiting...")
}

func simulateAppLogic() {
	LOG := kt_logging.GetLogger("main.thread")

	threadExecCount++
	LOG.Info("----- running round #%d", threadExecCount)

	// we received a message
	brokerTopic1_messageArrived.Inc()
	// let's simulate some "processing time"...
	processingMillis := 500 + rand.Intn(1500) // will be between 500 and 2000 millis
	LOG.Info("      simulating message pocessing took %d millis", processingMillis)
	brokerTopic1_processingTime.Observe(float64(processingMillis))

	if threadExecCount%3 == 0 {
		LOG.Info("      simulating message failed and was retried...")
		brokerTopic1_messageRetried.Inc()
		hasFailedEventually := rand.Intn(1000) < 500
		if hasFailedEventually {
			brokerTopic1_processingFailed.Inc()
			LOG.Info("      ... and eventually failed!")
		} else {
			LOG.Info("      ... but eventually succeeded!")
		}
	}

}
