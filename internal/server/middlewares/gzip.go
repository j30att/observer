package middlewares

import (
	"io"
	"mime"
	"net/http"
	"strings"

	"j30att/observer/internal/compression"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	writer        io.WriteCloser
	acceptsGzip   bool
	gzipEnabled   bool
	headerWritten bool
}

func newGzipResponseWriter(w http.ResponseWriter, acceptsGzip bool) *gzipResponseWriter {
	return &gzipResponseWriter{
		ResponseWriter: w,
		acceptsGzip:    acceptsGzip,
	}
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.headerWritten {
		return
	}

	w.headerWritten = true
	w.gzipEnabled = w.acceptsGzip && isCompressibleContentType(w.Header().Get("Content-Type"))
	if w.gzipEnabled {
		w.Header().Set("Content-Encoding", compression.GzipEncoding)
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		w.writer = compression.NewGzipWriter(w.ResponseWriter)
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}

	if !w.gzipEnabled {
		return w.ResponseWriter.Write(data)
	}

	_, err := w.writer.Write(data)
	if err != nil {
		return 0, err
	}

	return len(data), nil
}

func (w *gzipResponseWriter) Close() error {
	if w.writer == nil {
		return nil
	}

	return w.writer.Close()
}

func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasEncoding(r.Header.Get("Content-Encoding"), compression.GzipEncoding) {
			body, err := compression.NewGzipReadCloser(r.Body)
			if err != nil {
				http.Error(w, "invalid gzip body", http.StatusBadRequest)
				return
			}

			r.Body = body
		}

		gzipWriter := newGzipResponseWriter(w, hasEncoding(r.Header.Get("Accept-Encoding"), compression.GzipEncoding))
		defer func() {
			_ = gzipWriter.Close()
		}()

		next.ServeHTTP(gzipWriter, r)
	})
}

func hasEncoding(header string, encoding string) bool {
	for _, value := range strings.Split(header, ",") {
		encodingName, _, _ := strings.Cut(value, ";")
		if strings.EqualFold(strings.TrimSpace(encodingName), encoding) {
			return true
		}
	}

	return false
}

func isCompressibleContentType(contentType string) bool {
	if contentType == "" {
		return false
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}

	return mediaType == "application/json" || mediaType == "text/html"
}
