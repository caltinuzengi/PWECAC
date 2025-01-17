package main

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/caltinuzengi/pwecac/poolrescollector"
	"github.com/caltinuzengi/pwecac/resmoncollector"
)

func main() {

	// Resource Monitor Collector
	collector := resmoncollector.NewCombinedNetworkCollector()
	if err := prometheus.Register(collector); err != nil {
		log.Fatalf("Failed to register collector: %v", err)
	}

	// Pool Resource collector
	poolResCollector := poolrescollector.NewPoolResourceCollector()
	if err := prometheus.Register(poolResCollector); err != nil {
		log.Fatalf("Failed to register pool resource collector: %v", err)
	}

	http.Handle("/metrics", promhttp.Handler())

	addr := "0.0.0.0:9183"
	log.Printf("Starting Windows Network Exporter on %s/metrics\n", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}
