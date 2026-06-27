package server

import (
	"net/http"
	"strings"

	"github.com/wzhongyou/baize/server/middleware"
)

// routes builds the server's HTTP handler tree.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// ── API v1 ────────────────────────────────────────────────────────────
	mux.HandleFunc("/api/v1/health", s.handleHealth)
	mux.HandleFunc("/api/v1/chat", s.handleChat)
	mux.HandleFunc("/api/v1/tools/list", s.handleToolsList)
	mux.HandleFunc("/api/v1/tools/call", s.handleToolCall)
	mux.HandleFunc("/api/v1/sessions", s.handleSessions)
	mux.HandleFunc("/api/v1/sessions/", s.handleSessionByID)
	mux.HandleFunc("/api/v1/memory/search", s.handleMemorySearch)
	mux.HandleFunc("/api/v1/memory/save", s.handleMemorySave)

	// ── Middleware stack (outermost first) ────────────────────────────────
	var h http.Handler = mux
	h = middleware.RequestID(h)
	h = middleware.CORS(h)
	h = middleware.Logging(h)

	return h
}

// extractID extracts a resource ID from the URL path.
// e.g. "/api/v1/sessions/sess-123" → "sess-123"
func extractID(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	return strings.TrimSuffix(id, "/")
}
