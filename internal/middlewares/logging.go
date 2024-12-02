package middlewares

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"
)

func LoggingMiddleware(log *zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
			// Start time
			start := time.Now()

			// Read and store the raw body for logging
			var rawBody string
			if request.Body != nil {
				bodyBytes, err := io.ReadAll(request.Body)
				if err == nil {
					rawBody = string(bodyBytes)
					// Replace the request.Body to allow further handlers to read it
					request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}

			// Wrap the response writer to capture status code
			ww := middleware.NewWrapResponseWriter(response, request.ProtoMajor)

			defer func() {
				// Log the request details
				log.Info().
					Str("method", request.Method).
					Str("url", request.URL.String()).
					Int("status", ww.Status()).
					Int("bytes", ww.BytesWritten()).
					Str("remote", request.RemoteAddr).
					Dur("duration", time.Since(start)).
					Interface("body", rawBody).
					Msg("HTTP Request")
			}()

			// Call the next handler in the chain
			next.ServeHTTP(ww, request)
		})
	}
}
