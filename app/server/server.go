package server

import (
	"context"
	"fmt"
	"log"
	"my-dns-server/app/errors"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Server represents a DNS server
type Server struct {
	packetConn net.PacketConn
	shutdown   chan struct{}
	waitGroup  sync.WaitGroup
}

// NewServer creates a new DNS server instance
func NewServer(port int) (*Server, error) {
	address := fmt.Sprintf(":%d", port)
	log.Printf("INFO: Creating new DNS server on port %d", port)

	packetConn, err := net.ListenPacket("udp", address)
	if err != nil {
		log.Printf("ERROR: Failed to bind to port %d: %v", port, err)
		return nil, errors.Wrap(err, errors.ErrorTypeServer, fmt.Sprintf("failed to bind to port %d", port))
	}

	s := &Server{
		packetConn: packetConn,
		shutdown:   make(chan struct{}),
	}

	log.Printf("INFO: DNS server created successfully")
	return s, nil
}

// Run starts the server and listens for incoming DNS packets
func (s *Server) Run() error {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("INFO: Starting DNS server")
	// Start accepting connections
	go s.listenForPackets()

	// Wait for the shutdown signal
	<-sigChan
	log.Println("INFO: Shutdown signal received")
	return s.Shutdown()
}

// listenForPackets accepts incoming connections
func (s *Server) listenForPackets() {
	// DNS typically uses packets up to 512 bytes, but can be larger with EDNS.
	// A buffer of 1500 (common MTU) or 4096 should be safe.
	buffer := make([]byte, 4096)

	log.Printf("INFO: Listening for incoming packets")

	for {
		select {
		case <-s.shutdown:
			return
		default:
			n, addr, err := s.packetConn.ReadFrom(buffer)
			if err != nil {
				// Check if the error is due to the connection being closed during shutdown
				select {
				case <-s.shutdown:
					log.Println("DEBUG: Packet listener stopped due to shutdown")
					return
				default:
				}
				// For other errors, log and continue
				log.Printf("WARN: Failed to read packet: %v", err)
				continue
			}

			// Handle the received packet
			s.waitGroup.Add(1)
			request := make([]byte, n)
			copy(request, buffer[:n])
			log.Printf("DEBUG: Dispatching packet from %s for handling", addr)
			go s.handlePacket(request, addr)
		}
	}
}

// handlePacket processes a single DNS packet
func (s *Server) handlePacket(bytes []byte, addr net.Addr) {
	defer s.waitGroup.Done()

	log.Printf("DEBUG: Processing packet from %s", addr)
	log.Printf("DEBUG: Packet contents: %s", bytes)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	log.Println("INFO: Initiating server shutdown")

	// Signal all goroutines to stop
	close(s.shutdown)

	// Close the packet connection. This will unblock ReadFrom in listenForPackets.
	if s.packetConn != nil {
		if err := s.packetConn.Close(); err != nil {
			// Log the error but continue shutdown, as we still want to wait for goroutines.
			log.Printf("ERROR: Failed to close packet connection: %v", err)
		}
	}

	// Wait for all packet handlers to complete, with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Increased timeout slightly
	defer cancel()

	done := make(chan struct{})
	go func() {
		s.waitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("INFO: All active handlers finished. Server shutdown complete")
	case <-ctx.Done():
		log.Printf("ERROR: Server shutdown timed out waiting for handlers")
		return fmt.Errorf("server shutdown timed out") // Indicate timeout as an error
	}

	return nil
}
