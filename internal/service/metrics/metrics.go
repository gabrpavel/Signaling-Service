package metrics

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

func StartMetricsServer(address string) error {
	http.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(address, nil)
}
