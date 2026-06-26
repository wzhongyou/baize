package middleware

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

type contextKey string

const RequestIDKey contextKey = "request_id"

// RequestID returns middleware that injects a unique request ID into every request.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = generateID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID extracts the request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return fmt.Sprintf("req-%s", string(b))
}
