package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/wzhongyou/baize/protocol"
	"github.com/wzhongyou/baize/server/middleware"
)

func (s *Server) handleToolsList(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPost {
		protocol.WriteError(w, reqID, http.StatusMethodNotAllowed, protocol.CodeBadRequest, "method not allowed")
		return
	}

	if s.tools == nil {
		protocol.WriteSuccess(w, reqID, protocol.ListToolsResponse{Tools: []protocol.ToolInfo{}})
		return
	}

	infos := s.tools.ToolInfos()
	if infos == nil {
		infos = []protocol.ToolInfo{}
	}
	protocol.WriteSuccess(w, reqID, protocol.ListToolsResponse{Tools: infos})
}

func (s *Server) handleToolCall(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPost {
		protocol.WriteError(w, reqID, http.StatusMethodNotAllowed, protocol.CodeBadRequest, "method not allowed")
		return
	}

	var req protocol.CallToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		protocol.WriteError(w, reqID, http.StatusBadRequest, protocol.CodeBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		protocol.WriteError(w, reqID, http.StatusBadRequest, protocol.CodeBadRequest, "tool name is required")
		return
	}

	if s.tools == nil {
		protocol.WriteError(w, reqID, http.StatusNotFound, protocol.CodeNotFound, "no tools available")
		return
	}

	result, err := s.tools.Execute(context.Background(), req.Name, req.Arguments)
	if err != nil {
		protocol.WriteError(w, reqID, http.StatusInternalServerError, protocol.CodeToolError, err.Error())
		return
	}

	protocol.WriteSuccess(w, reqID, protocol.CallToolResponse{Content: result})
}
