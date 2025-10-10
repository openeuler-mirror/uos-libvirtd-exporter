package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

// SignalHandler handles OS signals for graceful shutdown
type SignalHandler struct {
	collector *LibvirtCollector
	sigChan   chan os.Signal
}

// NewSignalHandler creates a new signal handler
func NewSignalHandler(collector *LibvirtCollector) *SignalHandler {
	return &SignalHandler{
		collector: collector,
		sigChan:   make(chan os.Signal, 1),
	}
}

// Start starts listening for signals
func (s *SignalHandler) Start() {
	signal.Notify(s.sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-s.sigChan
		log.Println("Shutting down...")
		s.shutdown()
		os.Exit(0)
	}()
}

// shutdown performs cleanup operations
func (s *SignalHandler) shutdown() {
	if s.collector != nil {
		s.collector.Close()
	}
	log.Println("Shutdown complete")
}
