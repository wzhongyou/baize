package protocol

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ── Response helpers ────────────────────────────────────────────────────────

// WriteSuccess writes a standard success response.
func WriteSuccess(w http.ResponseWriter, requestID string, data any) {
	body := Response{
		Code:      CodeOK,
		RequestID: requestID,
	}
	if data != nil {
		raw, _ := json.Marshal(data)
		body.Data = raw
	}
	writeJSON(w, http.StatusOK, body)
}

// WriteCreated writes a 201 response.
func WriteCreated(w http.ResponseWriter, requestID string, data any) {
	body := Response{
		Code:      CodeOK,
		RequestID: requestID,
	}
	if data != nil {
		raw, _ := json.Marshal(data)
		body.Data = raw
	}
	writeJSON(w, http.StatusCreated, body)
}

// WriteError writes a standard error response.
func WriteError(w http.ResponseWriter, requestID string, httpStatus int, code int, message string) {
	body := Response{
		Code:      code,
		Message:   message,
		RequestID: requestID,
	}
	writeJSON(w, httpStatus, body)
}

// ── Error constructors ──────────────────────────────────────────────────────

// Errorf formats an API error with a code and HTTP status.
func Errorf(code int, format string, args ...any) *APIError {
	return &APIError{
		Code:       code,
		HTTPStatus: codeToHTTP(code),
		Message:    fmt.Sprintf(format, args...),
	}
}

// APIError is a typed error that maps to an HTTP response.
type APIError struct {
	Code       int
	HTTPStatus int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// WriteTo writes the error as a JSON response.
func (e *APIError) WriteTo(w http.ResponseWriter, requestID string) {
	WriteError(w, requestID, e.HTTPStatus, e.Code, e.Message)
}

// codeToHTTP maps API error codes to HTTP status codes.
func codeToHTTP(code int) int {
	switch {
	case code == CodeBadRequest:
		return http.StatusBadRequest
	case code == CodeUnauthorized:
		return http.StatusUnauthorized
	case code == CodeNotFound:
		return http.StatusNotFound
	case code == CodeConflict:
		return http.StatusConflict
	case code == CodeRateLimit:
		return http.StatusTooManyRequests
	case code == CodeTimeout:
		return http.StatusGatewayTimeout
	default:
		return http.StatusInternalServerError
	}
}

// ── Internal ────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
