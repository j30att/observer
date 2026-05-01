package middlewares

import (
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"strings"
)

type gzipResponseWriter struct {
	http.ResponseWriter
	writer        *gzip.Writer
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
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Add("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")
		w.writer = gzip.NewWriter(w.ResponseWriter)
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

type gzipReadCloser struct {
	gzipBody *gzip.Reader
	body     io.Closer
}

func (r *gzipReadCloser) Read(p []byte) (int, error) {
	return r.gzipBody.Read(p)
}

func (r *gzipReadCloser) Close() error {
	gzipErr := r.gzipBody.Close()
	bodyErr := r.body.Close()
	if gzipErr != nil {
		return gzipErr
	}

	return bodyErr
}

func Gzip(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasEncoding(r.Header.Get("Content-Encoding"), "gzip") {
			gzipBody, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "invalid gzip body", http.StatusBadRequest)
				return
			}

			r.Body = &gzipReadCloser{
				gzipBody: gzipBody,
				body:     r.Body,
			}
		}

		gzipWriter := newGzipResponseWriter(w, hasEncoding(r.Header.Get("Accept-Encoding"), "gzip"))
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
