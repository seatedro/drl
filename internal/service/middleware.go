// internal/service/middleware.go
package service

import (
	"log"
	"net/http"
	"time"
)

func debugMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status code
		wrapper := &responseWrapper{ResponseWriter: w}

		next.ServeHTTP(wrapper, r)

		log.Printf("[DEBUG] %s %s - Status: %d, Duration: %v\n",
			r.Method, r.URL.Path, wrapper.status, time.Since(start))
	})
}

type responseWrapper struct {
	http.ResponseWriter
	status int
}

func (rw *responseWrapper) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWrapper) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}
