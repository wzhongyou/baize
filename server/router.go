package server

import (
	"net/http"
	"strings"

	"github.com/wzhongyou/baize/server/middleware"
)

// routes builds the server's HTTP handler tree.
func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	// ── AGUI static files ─────────────────────────────────────────────────
	s.mountStatic(mux)

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

// mountStatic serves the AGUI web frontend if built, otherwise a placeholder.
func (s *Server) mountStatic(mux *http.ServeMux) {
	const aguiDir = "web/dist"
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// If a static file exists for this path, serve it.
		if _, err := http.Dir(aguiDir).Open(r.URL.Path); err == nil {
			http.FileServer(http.Dir(aguiDir)).ServeHTTP(w, r)
			return
		}
		// For SPA routing, serve index.html if it exists.
		if _, err := http.Dir(aguiDir).Open("index.html"); err == nil {
			http.ServeFile(w, r, aguiDir+"/index.html")
			return
		}
		// Otherwise placeholder.
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Write([]byte(placeholderHTML))
			return
		}
		http.NotFound(w, r)
	})
}

// extractID extracts a resource ID from the URL path.
// e.g. "/api/v1/sessions/sess-123" → "sess-123"
func extractID(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	return strings.TrimSuffix(id, "/")
}

const placeholderHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head><meta charset="UTF-8"><title>Baize AGUI</title></head>
<body style="font-family:system-ui;max-width:800px;margin:80px auto;padding:20px;background:#0f172a;color:#e2e8f0">
<h1>Baize AGUI</h1>
<p>AGUI Web 前端尚未构建。</p>
<pre style="background:#1e293b;padding:16px;border-radius:8px;color:#94a3b8">
cd web
npm install
npm run build
</pre>
<p>之后重启 <code>baize server</code> 即可访问完整界面。</p>
<p style="color:#64748b;margin-top:40px">API 端点：<br>
GET  /api/v1/health<br>
POST /api/v1/chat<br>
POST /api/v1/tools/list<br>
POST /api/v1/tools/call<br>
GET  /api/v1/sessions</p>
</body></html>`
