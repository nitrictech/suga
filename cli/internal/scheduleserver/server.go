package scheduleserver

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/nitrictech/suga/cli/internal/netx"
)

// Server provides HTTP endpoints to manually trigger schedules
type Server struct {
	services map[string]ServiceWithSchedules
	mux      *http.ServeMux
	listener net.Listener
	port     netx.ReservedPort
	server   *http.Server
}

// ServiceWithSchedules interface for services that support schedule triggering
type ServiceWithSchedules interface {
	GetName() string
	TriggerSchedule(index int, async bool) error
}

// NewServer creates a new schedule trigger server
func NewServer(services map[string]ServiceWithSchedules) (*Server, error) {
	// Get an available port
	port, err := netx.GetNextPort(netx.MinPort(8000), netx.MaxPort(8999))
	if err != nil {
		return nil, fmt.Errorf("failed to find open port: %w", err)
	}

	s := &Server{
		services: services,
		mux:      http.NewServeMux(),
		port:     port,
	}

	s.setupRoutes()

	return s, nil
}

func (s *Server) setupRoutes() {
	// Schedule trigger endpoint: GET /schedules/{serviceId}/{scheduleIndex}?async=true
	s.mux.HandleFunc("/schedules/{serviceId}/{scheduleIndex}", s.handleTriggerSchedule)
}

func (s *Server) handleTriggerSchedule(w http.ResponseWriter, r *http.Request) {
	// Only accept GET requests
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "✗ Method not allowed. Use GET to trigger schedules.\n")
		return
	}

	// Extract path parameters
	serviceId := r.PathValue("serviceId")
	scheduleIndexStr := r.PathValue("scheduleIndex")

	// Parse schedule index
	scheduleIndex, err := strconv.Atoi(scheduleIndexStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "✗ Invalid schedule index: %s\n", scheduleIndexStr)
		return
	}

	// Check if schedule index is valid (non-negative)
	if scheduleIndex < 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "✗ Schedule index must be non-negative\n")
		return
	}

	// Find the service
	svc, ok := s.services[serviceId]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "✗ Service '%s' not found\n", serviceId)
		return
	}

	// Parse async query parameter (defaults to false for synchronous execution)
	asyncStr := r.URL.Query().Get("async")
	async := false
	if asyncStr == "true" || asyncStr == "1" {
		async = true
	}

	// Trigger the schedule
	err = svc.TriggerSchedule(scheduleIndex, async)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "✗ Failed to trigger schedule %d on service '%s': %v\n", scheduleIndex, serviceId, err)
		return
	}

	// Success response
	w.WriteHeader(http.StatusOK)
	if async {
		fmt.Fprintf(w, "✓ Schedule %d on service '%s' triggered asynchronously\n", scheduleIndex, serviceId)
	} else {
		fmt.Fprintf(w, "✓ Schedule %d on service '%s' executed successfully\n", scheduleIndex, serviceId)
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf("localhost:%d", s.port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.listener = listener
	s.server = &http.Server{
		Handler: s.mux,
	}

	// Start server in goroutine
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Schedule trigger server error: %v\n", err)
		}
	}()

	return nil
}

// Stop stops the HTTP server
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// GetPort returns the port the server is listening on
func (s *Server) GetPort() int {
	return int(s.port)
}

// GetURL returns the base URL of the schedule trigger server
func (s *Server) GetURL() string {
	return fmt.Sprintf("http://localhost:%d", s.port)
}
