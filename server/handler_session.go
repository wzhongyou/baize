package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/wzhongyou/baize/api"
	"github.com/wzhongyou/baize/server/middleware"
	"github.com/wzhongyou/baize/session"
)

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	switch r.Method {
	case http.MethodGet:
		s.listSessions(w, reqID)
	case http.MethodPost:
		s.createSession(w, r, reqID)
	default:
		api.WriteError(w, reqID, http.StatusMethodNotAllowed, api.CodeBadRequest, "method not allowed")
	}
}

func (s *Server) listSessions(w http.ResponseWriter, reqID string) {
	sessions, err := s.sessions.ListSessions()
	if err != nil {
		api.WriteError(w, reqID, http.StatusInternalServerError, api.CodeInternalError, err.Error())
		return
	}

	items := make([]api.SessionInfo, 0, len(sessions))
	for _, sess := range sessions {
		items = append(items, api.SessionInfo{
			ID:            sess.ID,
			Title:         sess.Title,
			WorkspaceRoot: sess.WorkspaceRoot,
			Model:         sess.Model,
			StepCount:     sess.StepCount,
			TotalTokens:   sess.TotalTokens,
			Status:        string(sess.Status),
			CreatedAt:     sess.CreatedAt,
			UpdatedAt:     sess.UpdatedAt,
		})
	}
	if items == nil {
		items = []api.SessionInfo{}
	}

	api.WriteSuccess(w, reqID, map[string]any{"sessions": items})
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request, reqID string) {
	var req api.CreateSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, reqID, http.StatusBadRequest, api.CodeBadRequest, "invalid request body")
		return
	}

	id := fmt.Sprintf("sess-%d", time.Now().UnixNano())
	if err := s.sessions.CreateSession(&session.Session{
		ID:            id,
		Title:         req.Title,
		WorkspaceRoot: req.WorkspaceRoot,
	}); err != nil {
		api.WriteError(w, reqID, http.StatusInternalServerError, api.CodeInternalError, err.Error())
		return
	}

	api.WriteCreated(w, reqID, map[string]string{"id": id})
}

func (s *Server) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())
	id := extractID(r.URL.Path, "/api/v1/sessions/")
	if id == "" {
		api.WriteError(w, reqID, http.StatusBadRequest, api.CodeBadRequest, "session id required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.getSession(w, reqID, id)
	case http.MethodDelete:
		s.deleteSession(w, reqID, id)
	default:
		api.WriteError(w, reqID, http.StatusMethodNotAllowed, api.CodeBadRequest, "method not allowed")
	}
}

func (s *Server) getSession(w http.ResponseWriter, reqID, id string) {
	sess, err := s.sessions.GetSession(id)
	if err != nil {
		api.WriteError(w, reqID, http.StatusNotFound, api.CodeNotFound, "session not found")
		return
	}

	msgs := make([]api.Message, 0, len(sess.Messages))
	for _, m := range sess.Messages {
		apiMsg := api.Message{
			Role:      string(m.Role),
			Content:   m.Content,
			ToolName:  m.ToolName,
			Timestamp: m.Timestamp,
		}
		for _, tc := range m.ToolCalls {
			apiMsg.ToolCalls = append(apiMsg.ToolCalls, api.ToolCall{
				ID:        tc.ID,
				Name:      tc.Name,
				Arguments: tc.Arguments,
			})
		}
		msgs = append(msgs, apiMsg)
	}

	api.WriteSuccess(w, reqID, api.SessionDetail{
		SessionInfo: api.SessionInfo{
			ID:            sess.ID,
			Title:         sess.Title,
			WorkspaceRoot: sess.WorkspaceRoot,
			Model:         sess.Model,
			StepCount:     sess.StepCount,
			TotalTokens:   sess.TotalTokens,
			Status:        string(sess.Status),
			CreatedAt:     sess.CreatedAt,
			UpdatedAt:     sess.UpdatedAt,
		},
		Messages: msgs,
	})
}

func (s *Server) deleteSession(w http.ResponseWriter, reqID, id string) {
	if err := s.sessions.DeleteSession(id); err != nil {
		api.WriteError(w, reqID, http.StatusInternalServerError, api.CodeInternalError, err.Error())
		return
	}
	api.WriteSuccess(w, reqID, map[string]string{"status": "deleted"})
}
