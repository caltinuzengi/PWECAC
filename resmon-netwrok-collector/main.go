package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	collector := NewCombinedNetworkCollector()
	if err := prometheus.Register(collector); err != nil {
		log.Fatalf("Failed to register collector: %v", err)
	}

	http.Handle("/metrics", promhttp.Handler())

	addr := "0.0.0.0:9183"
	log.Printf("Starting Windows Network Exporter on %s/metrics\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}
