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
	listener    net.Listener
	connections sync.Map
	shutdown    chan struct{}
	waitGroup   sync.WaitGroup
}

func NewServer(port int) (*Server, error) {
	listener, err := net.Listen("udp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		return nil, errors.Wrap(err, errors.ErrorTypeServer, fmt.Sprintf("failed to bind to port %d", port))
	}

	s := &Server{
		listener: listener,
		shutdown: make(chan struct{}),
	}

	return s, nil
}

// Run starts the server and listens for connections
func (s *Server) Run() error {
	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start accepting connections
	go s.acceptConnections()

	// Wait for the shutdown signal
	<-sigChan
	return s.Shutdown()
}

// acceptConnections accepts incoming connections
func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.shutdown:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			s.waitGroup.Add(1)
			s.connections.Store(conn.RemoteAddr(), conn)
		}
	}
}

// handleConnection handles a client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		// Cleanup
		err := conn.Close()
		if err != nil {
			log.Printf("Failed to close connection: %v", err)
		}
		s.connections.Delete(conn.RemoteAddr())
		s.waitGroup.Done()
	}()

	// TODO: Logic to handle incoming connection later
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	// Signal shutdown
	close(s.shutdown)

	// Close listener to stop accepting new connections
	if err := s.listener.Close(); err != nil {
		return errors.Wrap(err, errors.ErrorTypeServer, "failed to close listener")
	}

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Wait for all connections to close or timeout
	done := make(chan struct{})
	go func() {
		s.waitGroup.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("Server shutdown complete")
	case <-ctx.Done():
		log.Printf("Server shutdown timed out")
	}

	return nil
}
