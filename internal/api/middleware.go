package api

import (
	"net/http"
	"strings"
)

type contextKey string

const requestIDKey contextKey = "request_id"

func requestMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		if strings.HasPrefix(r.URL.Path, "/v1/") && r.Method != "GET" {
			id := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
			if id == "" {
				writeError(w, r, &requestError{Message: "必须提供 Idempotency-Key 请求头", Status: 400})
				return
			}
			if len(id) > 128 {
				writeError(w, r, &requestError{Message: "Idempotency-Key 过长", Status: 400})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func requestID(r *http.Request) string { return strings.TrimSpace(r.Header.Get("Idempotency-Key")) }
