package server

import (
	"log"
	"net/http"
)

// Server is the Abalone HTTP and WebSocket server
type Server struct {
	rooms *RoomManager
	webFS http.FileSystem
	mux   *http.ServeMux
}

// NewServer creates a new server with the given filesystem for static assets
func NewServer(webFS http.FileSystem) *Server {
	s := &Server{
		rooms: NewRoomManager(),
		webFS: webFS,
		mux:   http.NewServeMux(),
	}
	s.mux.HandleFunc("/ws", s.handleWebSocket)
	s.mux.Handle("/", http.FileServer(webFS))
	return s
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

// ListenAndServe starts the HTTP server on the given address
func (s *Server) ListenAndServe(addr string) error {
	log.Printf("Abalone server starting on %s", addr)
	return http.ListenAndServe(addr, s)
}
