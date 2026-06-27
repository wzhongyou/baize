package server

import (
	"net/http"

	"github.com/wzhongyou/baize/protocol"
	"github.com/wzhongyou/baize/server/middleware"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		protocol.WriteError(w, middleware.GetRequestID(r.Context()), http.StatusMethodNotAllowed, protocol.CodeBadRequest, "method not allowed")
		return
	}
	protocol.WriteSuccess(w, middleware.GetRequestID(r.Context()), map[string]any{
		"status":  "ok",
		"version": "0.3.0",
	})
}
