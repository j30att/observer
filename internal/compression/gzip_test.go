package compression_test

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/compression"
)

func TestGzipCompression(t *testing.T) {
	t.Run("Тест gzip helpers", func(t *testing.T) {
		t.Run("Должен сжать тело и вернуть читаемый gzip reader", func(t *testing.T) {
			body := []byte(`{"id":"Alloc","type":"gauge","value":12.5}`)

			compressed, err := compression.CompressGzip(body)
			require.NoError(t, err)

			reader, err := compression.NewGzipReadCloser(io.NopCloser(bytes.NewReader(compressed)))
			require.NoError(t, err)
			defer func() {
				_ = reader.Close()
			}()

			decompressed, err := io.ReadAll(reader)

			require.NoError(t, err)
			assert.Equal(t, body, decompressed)
		})

		t.Run("Должен вернуть ошибку для невалидного gzip тела", func(t *testing.T) {
			_, err := compression.NewGzipReadCloser(io.NopCloser(bytes.NewReader([]byte("not gzip"))))

			require.Error(t, err)
		})
	})
}
