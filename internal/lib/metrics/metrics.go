package lib

import (
	"context"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var requestMetrics = promauto.NewSummaryVec(prometheus.SummaryOpts{
	Namespace:  "signaling",
	Subsystem:  "websocket",
	Name:       "request",
	Help:       "Response time of HTTP requests.",
	Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
}, []string{"status"})

func ObserveRequest(d time.Duration, status int) {
	requestMetrics.WithLabelValues(strconv.Itoa(status)).Observe(d.Seconds())
}

func PrometheusInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)

	st, _ := status.FromError(err)
	ObserveRequest(duration, int(st.Code()))

	return resp, err
}
