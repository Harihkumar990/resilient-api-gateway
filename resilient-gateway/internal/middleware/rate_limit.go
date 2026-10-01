package middleware

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"fmt"
	"resilient-gateway/internal/limiter"
)

func RateLimitMiddleware(rl *limiter.RateLimiter) func (http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)
			result, err := rl.Allow(r.Context(), clientIP)
			if err != nil {
				http.Error(w, "Rate limiter error", http.StatusInternalServerError)
				return
			}
			w.Header().Set("X-Ratelimit-Remaining", strconv.Itoa(result.Remaining))
			w.Header().Set("X-Rated-Reset", fmt.Sprintf("%ds",int(result.Reset.Seconds())))
			if !result.Allowed {
				RateLimitBlocks.WithLabelValues(clientIP).Inc()
				w.Header().Set("Retry-After", strconv.Itoa(int(result.Reset.Seconds())))
				http.Error(w, "too many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w,r)
		})
	}
}

func getClientIP(r *http.Request) string {
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}