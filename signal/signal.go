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
	collector *collector.LibvirtCollector
	sigChan   chan os.Signal
}

// NewHandler creates a new signal handler
func NewHandler(collector *collector.LibvirtCollector) *Handler {
	return &Handler{
		collector: collector,
		sigChan:   make(chan os.Signal, 1),
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
	if s.collector != nil {
		s.collector.Close()
	}
	log.Println("Shutdown complete")
}
