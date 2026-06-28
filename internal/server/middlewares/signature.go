package middlewares

import (
	"bytes"
	"io"
	"net/http"

	"j30att/observer/internal/signature"
)

type signatureResponseWriter struct {
	header      http.Header
	body        bytes.Buffer
	statusCode  int
	wroteHeader bool
}

func newSignatureResponseWriter() *signatureResponseWriter {
	return &signatureResponseWriter{
		header: make(http.Header),
	}
}

func (w *signatureResponseWriter) Header() http.Header {
	return w.header
}

func (w *signatureResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}

	w.statusCode = statusCode
	w.wroteHeader = true
}

func (w *signatureResponseWriter) Write(data []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	return w.body.Write(data)
}

// Signature verifies signed request bodies and signs response bodies.
func Signature(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestBody := r.Body
			if requestBody == nil {
				requestBody = http.NoBody
			}

			body, err := io.ReadAll(requestBody)
			if err != nil {
				writeSignedError(w, "failed to read request body", http.StatusBadRequest, key)
				return
			}
			_ = requestBody.Close()
			r.Body = io.NopCloser(bytes.NewReader(body))

			requestSignature := r.Header.Get(signature.Header)
			if requestSignature != "" && !signature.Verify(body, key, requestSignature) {
				writeSignedError(w, "invalid request signature", http.StatusBadRequest, key)
				return
			}

			recorder := newSignatureResponseWriter()
			next.ServeHTTP(recorder, r)
			writeSignedResponse(w, recorder.header, recorder.statusCode, recorder.body.Bytes(), key)
		})
	}
}

func writeSignedError(w http.ResponseWriter, message string, statusCode int, key string) {
	header := make(http.Header)
	header.Set("Content-Type", "text/plain; charset=utf-8")
	header.Set("X-Content-Type-Options", "nosniff")
	writeSignedResponse(w, header, statusCode, []byte(message+"\n"), key)
}

func writeSignedResponse(w http.ResponseWriter, header http.Header, statusCode int, body []byte, key string) {
	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	for name, values := range header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.Header().Set(signature.Header, signature.Sign(body, key))
	w.WriteHeader(statusCode)
	_, _ = w.Write(body)
}
