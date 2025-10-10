package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	libvirtURI  = flag.String("libvirt.uri", "qemu:///system", "Libvirt connection URI")
	listenAddr  = flag.String("web.listen-address", ":9177", "Address to listen on for web interface and telemetry")
	metricsPath = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
)

func main() {
	flag.Parse()

	log.Printf("Starting uos-libvirtd-exporter %s", version)
	log.Printf("Libvirt URI: %s", *libvirtURI)
	log.Printf("Listening on: %s", *listenAddr)
	log.Printf("Metrics path: %s", *metricsPath)

	// Create libvirt collector
	collector, err := NewLibvirtCollector(*libvirtURI)
	if err != nil {
		log.Fatalf("Failed to create libvirt collector: %v", err)
	}

	// Register collector
	prometheus.MustRegister(collector)

	// Setup HTTP handlers
	http.Handle(*metricsPath, promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>UOS Libvirt Exporter</title></head>
			<body>
			<h1>UOS Libvirt Exporter</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			<p>Build version: ` + version + `</p>
			</body>
			</html>`))
	})

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		collector.Close()
		os.Exit(0)
	}()

	// Start HTTP server
	log.Printf("Starting HTTP server on %s", *listenAddr)
	if err := http.ListenAndServe(*listenAddr, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}

var version = "dev"
