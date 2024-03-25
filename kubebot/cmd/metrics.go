package cmd

import (
	"log"
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Оголошення кастомних метрик
var (
	totalRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slackbot_total_requests",
			Help: "Total number of requests processed by the Slack bot.",
		},
		[]string{"path"},
	)
	totalErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "slackbot_total_errors",
			Help: "Total number of errors encountered by the Slack bot.",
		},
		[]string{"path"},
	)
)

func init() {
	// Реєстрація кастомних метрик у реєстрі Prometheus
	prometheus.MustRegister(totalRequests)
	prometheus.MustRegister(totalErrors)
}

func startMetricsServer() {
	// Налаштування хендлера для ендпоінта `/metrics`
	http.Handle("/metrics", promhttp.Handler())
	port := ":9090"
	log.Printf("Starting metrics server on %s", port)

	// Запуск HTTP сервера
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Failed to start metrics server: %v", err)
	}
}