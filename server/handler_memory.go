package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/wzhongyou/baize/protocol"
	"github.com/wzhongyou/baize/server/middleware"
)

func (s *Server) handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPost {
		protocol.WriteError(w, reqID, http.StatusMethodNotAllowed, protocol.CodeBadRequest, "method not allowed")
		return
	}

	var req protocol.MemorySearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		protocol.WriteError(w, reqID, http.StatusBadRequest, protocol.CodeBadRequest, "invalid request body")
		return
	}
	if req.Query == "" {
		protocol.WriteError(w, reqID, http.StatusBadRequest, protocol.CodeBadRequest, "query is required")
		return
	}
	if req.TopK <= 0 {
		req.TopK = 5
	}

	if s.memory == nil {
		protocol.WriteSuccess(w, reqID, protocol.MemorySearchResponse{Results: []protocol.MemorySearchResult{}})
		return
	}

	results, err := s.memory.Search(context.Background(), req.Query, req.TopK)
	if err != nil {
		protocol.WriteError(w, reqID, http.StatusInternalServerError, protocol.CodeInternalError, err.Error())
		return
	}

	out := make([]protocol.MemorySearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, protocol.MemorySearchResult{
			Content:  r.Content,
			Score:    r.Score,
			Metadata: r.Metadata,
		})
	}
	if out == nil {
		out = []protocol.MemorySearchResult{}
	}
	protocol.WriteSuccess(w, reqID, protocol.MemorySearchResponse{Results: out})
}

func (s *Server) handleMemorySave(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPost {
		protocol.WriteError(w, reqID, http.StatusMethodNotAllowed, protocol.CodeBadRequest, "method not allowed")
		return
	}

	var req protocol.MemorySaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		protocol.WriteError(w, reqID, http.StatusBadRequest, protocol.CodeBadRequest, "invalid request body")
		return
	}
	if req.Content == "" {
		protocol.WriteError(w, reqID, http.StatusBadRequest, protocol.CodeBadRequest, "content is required")
		return
	}

	if s.memory == nil {
		protocol.WriteError(w, reqID, http.StatusInternalServerError, protocol.CodeInternalError, "memory not configured")
		return
	}

	if err := s.memory.Save(context.Background(), req.Content, req.Metadata); err != nil {
		protocol.WriteError(w, reqID, http.StatusInternalServerError, protocol.CodeInternalError, err.Error())
		return
	}

	protocol.WriteSuccess(w, reqID, map[string]string{"status": "saved"})
}
