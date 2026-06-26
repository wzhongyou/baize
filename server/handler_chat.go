package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/wzhongyou/baize/agent"
	"github.com/wzhongyou/baize/api"
	"github.com/wzhongyou/baize/server/middleware"
	"github.com/wzhongyou/baize/session"
)

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.WriteError(w, middleware.GetRequestID(r.Context()), http.StatusMethodNotAllowed, api.CodeBadRequest, "method not allowed")
		return
	}

	var req api.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, middleware.GetRequestID(r.Context()), http.StatusBadRequest, api.CodeBadRequest, "invalid request body")
		return
	}
	if req.Message == "" {
		api.WriteError(w, middleware.GetRequestID(r.Context()), http.StatusBadRequest, api.CodeBadRequest, "message is required")
		return
	}
	if req.MaxSteps <= 0 {
		req.MaxSteps = 30
	}

	// Check streaming support.
	flusher, ok := w.(http.Flusher)
	if !ok {
		api.WriteError(w, middleware.GetRequestID(r.Context()), http.StatusInternalServerError, api.CodeInternalError, "streaming not supported")
		return
	}

	// SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Request-ID", middleware.GetRequestID(r.Context()))

	sendSSE := func(event api.ChatEvent) {
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Create or load session.
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = fmt.Sprintf("sess-%d", time.Now().UnixNano())
		title := req.Message
		if len(title) > 50 {
			title = title[:50] + "..."
		}
		_ = s.sessions.CreateSession(&session.Session{
			ID:    sessionID,
			Title: title,
		})
	}

	// Load existing messages.
	messages := []agent.Message{}
	if sess, err := s.sessions.GetSession(sessionID); err == nil {
		messages = sess.Messages
	}

	userMsg := agent.Message{
		Role:      agent.RoleUser,
		Content:   req.Message,
		Timestamp: time.Now(),
	}
	_ = s.sessions.AddMessage(sessionID, userMsg)
	messages = append(messages, userMsg)

	// Stream agent execution.
	var assistantContent strings.Builder

	s.agent.RunStream(r.Context(), AgentRunRequest{
		SessionID: sessionID,
		Message:   req.Message,
		Provider:  req.Provider,
		Model:     req.Model,
		MaxSteps:  req.MaxSteps,
	}, func(ev StreamEvent) {
		// Map to API event.
		apiEvent := api.ChatEvent{
			Type:     ev.Type,
			Content:  ev.Content,
			ToolName: ev.ToolName,
			ToolArgs: ev.ToolArgs,
			Tokens:   ev.Tokens,
		}
		sendSSE(apiEvent)

		// Track assistant content for saving.
		if ev.Type == api.EventAnswer {
			assistantContent.WriteString(ev.Content)
		}

		// Save assistant message on completion.
		if ev.Type == api.EventDone {
			if assistantContent.Len() > 0 {
				_ = s.sessions.AddMessage(sessionID, agent.Message{
					Role:      agent.RoleAssistant,
					Content:   assistantContent.String(),
					Timestamp: time.Now(),
				})
			}
		}
	})
}
