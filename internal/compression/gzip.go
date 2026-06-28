package compression

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// GzipEncoding is the HTTP content-coding token for gzip payloads.
const GzipEncoding = "gzip"

// CompressGzip returns body compressed with gzip.
func CompressGzip(body []byte) ([]byte, error) {
	var compressed bytes.Buffer

	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(body); err != nil {
		_ = writer.Close()
		return nil, fmt.Errorf("write gzip body: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}

	return compressed.Bytes(), nil
}

// NewGzipWriter wraps writer with a gzip compressor.
func NewGzipWriter(writer io.Writer) io.WriteCloser {
	return gzip.NewWriter(writer)
}

// NewGzipReadCloser wraps body with a gzip reader and closes both readers.
func NewGzipReadCloser(body io.ReadCloser) (io.ReadCloser, error) {
	gzipBody, err := gzip.NewReader(body)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}

	return &gzipReadCloser{
		gzipBody: gzipBody,
		body:     body,
	}, nil
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
