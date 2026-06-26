package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/wzhongyou/baize/api"
	"github.com/wzhongyou/baize/server/middleware"
)

func (s *Server) handleToolsList(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPost {
		api.WriteError(w, reqID, http.StatusMethodNotAllowed, api.CodeBadRequest, "method not allowed")
		return
	}

	if s.tools == nil {
		api.WriteSuccess(w, reqID, api.ListToolsResponse{Tools: []api.ToolInfo{}})
		return
	}

	infos := s.tools.ToolInfos()
	if infos == nil {
		infos = []api.ToolInfo{}
	}
	api.WriteSuccess(w, reqID, api.ListToolsResponse{Tools: infos})
}

func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPost {
		api.WriteError(w, reqID, http.StatusMethodNotAllowed, api.CodeBadRequest, "method not allowed")
		return
	}

	var req api.CallToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, reqID, http.StatusBadRequest, api.CodeBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		api.WriteError(w, reqID, http.StatusBadRequest, api.CodeBadRequest, "tool name is required")
		return
	}

	if s.tools == nil {
		api.WriteError(w, reqID, http.StatusNotFound, api.CodeNotFound, "no tools available")
		return
	}

	result, err := s.tools.Execute(context.Background(), req.Name, req.Arguments)
	if err != nil {
		api.WriteError(w, reqID, http.StatusInternalServerError, api.CodeToolError, err.Error())
		return
	}

	api.WriteSuccess(w, reqID, api.CallToolResponse{Content: result})
}
