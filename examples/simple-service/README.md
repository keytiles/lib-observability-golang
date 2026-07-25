# simple-service example

A small runnable demo of `lib-observability-golang`: global log/metric labels, predefined metric templates, HTTP server lazy metrics, and a Prometheus scrape endpoint.

## Run

From this directory:

```
go run .
```

Then:

- Successful HTTP: `http://localhost:8080/api/v1/ping`
- Failed HTTP: `http://localhost:8080/api/v1/ping-fail`
- Prometheus metrics: `http://localhost:9008/metrics`

Stop with Ctrl+C.
