package server

import (
	"context"
	"net/http"
	"time"
)

// Server wraps an http.Server with lifecycle helpers.
type Server struct {
	httpServer *http.Server
}

// New constructs a server ready to serve the provided handler on addr.
func New(addr string, readHeaderTimeout, idleTimeout time.Duration, handler http.Handler) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: readHeaderTimeout,
			IdleTimeout:       idleTimeout,
		},
	}
}

// Start begins listening for HTTP requests in a blocking manner.
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
