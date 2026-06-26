package server

import (
	"net/http"

	"github.com/wzhongyou/baize/api"
	"github.com/wzhongyou/baize/server/middleware"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.WriteError(w, middleware.GetRequestID(r.Context()), http.StatusMethodNotAllowed, api.CodeBadRequest, "method not allowed")
		return
	}
	api.WriteSuccess(w, middleware.GetRequestID(r.Context()), map[string]any{
		"status":  "ok",
		"version": "0.3.0",
	})
}
