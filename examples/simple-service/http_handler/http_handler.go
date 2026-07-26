package http_handler

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/keytiles/lib-logging-golang/v2/pkg/kt_logging"
	"github.com/keytiles/lib-observability-golang/v2/pkg/kt_observability_monitoring"
)

type HttpServerHandler struct {
	pingMetricSet *kt_observability_monitoring.HttpServerLazyMetricsSet
	logger        *kt_logging.Logger
}

func NewHttpServerHandler() *HttpServerHandler {
	serverId := "HttpServerHandler"
	return &HttpServerHandler{
		pingMetricSet: kt_observability_monitoring.NewHttpServerLazyMetricsSet("ping", kt_observability_monitoring.WithHttpServerId(serverId)),
		logger:        kt_logging.GetLogger("main.handler"),
	}
}

func (n *HttpServerHandler) ServePingOK(w http.ResponseWriter, req *http.Request) {
	startedAt := time.Now().UnixMilli()

	statusCode := 200
	body := "Pong"

	// we have a request - mark it
	n.pingMetricSet.ServeStarted(req)

	defer func() {
		// adjust metrics
		millisTook := time.Now().UnixMilli() - startedAt
		n.pingMetricSet.ServeTookMillis(req, strconv.Itoa(statusCode), float64(millisTook))
		n.pingMetricSet.ServeSucceeded(req, strconv.Itoa(statusCode))
	}()

	// let's wait some random time - simulating execution time
	delayMillis := 50 + rand.Intn(500)
	delay := time.Duration(delayMillis) * time.Millisecond
	time.Sleep(delay)

	n.logger.Info("Ping request served with \"pong\" - took %d millis", delayMillis)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
	w.Write([]byte(body))
}

func (n *HttpServerHandler) ServePingFailed(w http.ResponseWriter, req *http.Request) {
	startedAt := time.Now().UnixMilli()

	statusCode := 500
	body := "Oops, serving Ping failed"

	// we have a request - mark it
	n.pingMetricSet.ServeStarted(req)

	defer func() {
		// adjust metrics
		millisTook := time.Now().UnixMilli() - startedAt
		n.pingMetricSet.ServeTookMillis(req, strconv.Itoa(statusCode), float64(millisTook))
		n.pingMetricSet.ServeFailed(req, strconv.Itoa(statusCode))
	}()

	// let's wait some random time - simulating execution time
	delayMillis := 50 + rand.Intn(500)
	delay := time.Duration(delayMillis) * time.Millisecond
	time.Sleep(delay)

	n.logger.Info("Ping request served with failure - took %d millis", delayMillis)

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(statusCode)
	w.Write([]byte(body))
}
