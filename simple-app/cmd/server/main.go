package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"log"
	rand "math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
var simpleCounter prometheus.Counter
var simpleGauge prometheus.Gauge
var simpleBadRequestGauge prometheus.Gauge
var simpleStatusCodes *prometheus.CounterVec
var simpleProcessingTimeSummaryMs prometheus.Summary
var simpleProcessingTimeHistogramMs prometheus.Histogram
var httpDuration *prometheus.HistogramVec

func main() {
	flag.Parse()

	simpleCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Help: "simple_app_counter",
			Name: "simple_app_counter",
		})

	simpleGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "simple_app_gauge",
			Name: "simple_app_gauge",
		})

	simpleBadRequestGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Help: "simple_app_bad_request_gauge",
			Name: "simple_app_bad_request_gauge",
		})

	simpleStatusCodes = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "simple_app_status_codes",
			Help: "simple_app_status_codes",
		},
		[]string{"code", "method"})

	simpleProcessingTimeSummaryMs = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Name:       "simple_app_time_summary_ms",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		})

	simpleProcessingTimeHistogramMs = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "simple_processing_time_histogram_ms",
			Buckets: prometheus.LinearBuckets(0, 10, 20),
		})

	httpDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "simple_app_http_request_duration_seconds",
		Help: "Duration of HTTP requests.",
	}, []string{"path"})

	// Create non-global registry.
	reg := prometheus.NewRegistry()

	// Add go runtime metrics and process collectors.
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		simpleCounter,
		simpleGauge,
		simpleStatusCodes,
		simpleProcessingTimeSummaryMs,
		simpleProcessingTimeHistogramMs,
		simpleBadRequestGauge,
		httpDuration,
	)

	go func() {
		for {
			simpleCounter.Inc()
			time.Sleep(1000 * time.Millisecond)
		}
	}()

	go func() {
		src := rand.NewSource(time.Now().UnixNano())
		rnd := rand.New(src)
		for {
			obs := float64(100 + rnd.Intn(30))
			simpleProcessingTimeSummaryMs.Observe(obs)
			simpleProcessingTimeHistogramMs.Observe(obs)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Expose /metrics HTTP endpoint using the created custom registry.
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}))
	http.Handle("/gauge", PrometheusHTTPDurationMiddleware(handleGauge))
	http.Handle("/bad", PrometheusHTTPDurationMiddleware(handleBadRequest))
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func handleGauge(w http.ResponseWriter, r *http.Request) {
	//src := rand.NewSource(time.Now().UnixNano())
	//rnd := rand.New(src)
	time.Sleep(time.Millisecond * time.Duration(rand.Uint64()%1000))
	simpleGauge.Inc()
	simpleStatusCodes.WithLabelValues("200", "GET").Inc()
	w.WriteHeader(http.StatusOK)
}

func handleBadRequest(w http.ResponseWriter, r *http.Request) {
	simpleBadRequestGauge.Inc()
	simpleStatusCodes.WithLabelValues("400", "GET").Inc()
	w.WriteHeader(http.StatusBadRequest)
}

func PrometheusHTTPDurationMiddleware(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := strings.Trim(r.RequestURI, "/")
		timer := prometheus.NewTimer(httpDuration.WithLabelValues(route))
		next.ServeHTTP(w, r)
		timer.ObserveDuration()
	})
}
