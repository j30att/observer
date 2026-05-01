package middlewares

import (
	"net/http"
	"time"

	"github.com/rs/zerolog"
)

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode    int
	responseSize  int
	headerWritten bool
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{
		ResponseWriter: w,
	}
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}

	w.headerWritten = true
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *loggingResponseWriter) Write(data []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}

	size, err := w.ResponseWriter.Write(data)
	w.responseSize += size
	return size, err
}

func Logger(logger zerolog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			lrw := newLoggingResponseWriter(w)

			next.ServeHTTP(lrw, r)

			statusCode := lrw.statusCode
			if !lrw.headerWritten {
				statusCode = http.StatusOK
			}

			logger.Info().
				Str("uri", r.RequestURI).
				Str("method", r.Method).
				Dur("duration", time.Since(start)).
				Int("status", statusCode).
				Int("response_size", lrw.responseSize).
				Msg("handled request")
		})
	}
}
