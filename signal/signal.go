package signal

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gitee.com/openeuler/uos-libvirtd-exporter/collector"
)

// Handler handles OS signals for graceful shutdown
type Handler struct {
	exporter *collector.Exporter
	sigChan  chan os.Signal
}

// NewHandler creates a new signal handler
func NewHandler(exporter *collector.Exporter) *Handler {
	return &Handler{
		exporter: exporter,
		sigChan:  make(chan os.Signal, 1),
	}
}

// Start starts listening for signals
func (s *Handler) Start() {
	signal.Notify(s.sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-s.sigChan
		log.Println("Shutting down...")
		s.shutdown()
		os.Exit(0)
	}()
}

// shutdown performs cleanup operations
func (s *Handler) shutdown() {
	if s.exporter != nil {
		s.exporter.Close()
	}
	log.Println("Shutdown complete")
}