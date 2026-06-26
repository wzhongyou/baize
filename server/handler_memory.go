package server

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/wzhongyou/baize/api"
	"github.com/wzhongyou/baize/server/middleware"
)

func (s *Server) handleMemorySearch(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPost {
		api.WriteError(w, reqID, http.StatusMethodNotAllowed, api.CodeBadRequest, "method not allowed")
		return
	}

	var req api.MemorySearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, reqID, http.StatusBadRequest, api.CodeBadRequest, "invalid request body")
		return
	}
	if req.Query == "" {
		api.WriteError(w, reqID, http.StatusBadRequest, api.CodeBadRequest, "query is required")
		return
	}
	if req.TopK <= 0 {
		req.TopK = 5
	}

	if s.memory == nil {
		api.WriteSuccess(w, reqID, api.MemorySearchResponse{Results: []api.MemorySearchResult{}})
		return
	}

	results, err := s.memory.Search(context.Background(), req.Query, req.TopK)
	if err != nil {
		api.WriteError(w, reqID, http.StatusInternalServerError, api.CodeInternalError, err.Error())
		return
	}

	out := make([]api.MemorySearchResult, 0, len(results))
	for _, r := range results {
		out = append(out, api.MemorySearchResult{
			Content:  r.Content,
			Score:    r.Score,
			Metadata: r.Metadata,
		})
	}
	if out == nil {
		out = []api.MemorySearchResult{}
	}
	api.WriteSuccess(w, reqID, api.MemorySearchResponse{Results: out})
}

func (s *Server) handleMemorySave(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPost {
		api.WriteError(w, reqID, http.StatusMethodNotAllowed, api.CodeBadRequest, "method not allowed")
		return
	}

	var req api.MemorySaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, reqID, http.StatusBadRequest, api.CodeBadRequest, "invalid request body")
		return
	}
	if req.Content == "" {
		api.WriteError(w, reqID, http.StatusBadRequest, api.CodeBadRequest, "content is required")
		return
	}

	if s.memory == nil {
		api.WriteError(w, reqID, http.StatusInternalServerError, api.CodeInternalError, "memory not configured")
		return
	}

	if err := s.memory.Save(context.Background(), req.Content, req.Metadata); err != nil {
		api.WriteError(w, reqID, http.StatusInternalServerError, api.CodeInternalError, err.Error())
		return
	}

	api.WriteSuccess(w, reqID, map[string]string{"status": "saved"})
}
