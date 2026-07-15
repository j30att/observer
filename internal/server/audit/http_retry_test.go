package audit

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRetryableHTTPClientRetriesTransportErrors(t *testing.T) {
	attempts := 0
	bodies := make([]string, 0, 2)
	client := newRetryableHTTPClient(
		&http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				attempts++

				body, err := io.ReadAll(req.Body)
				require.NoError(t, err)
				bodies = append(bodies, string(body))

				if attempts == 1 {
					return nil, temporaryNetError{}
				}

				return &http.Response{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(bytes.NewReader(nil)),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			}),
		},
		[]time.Duration{0},
	)
	req, err := http.NewRequest(http.MethodPost, "http://example.test/audit", bytes.NewReader([]byte(`{"id":"Alloc"}`)))
	require.NoError(t, err)

	resp, err := client.Do(req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusAccepted, resp.StatusCode)
	assert.Equal(t, 2, attempts)
	assert.Equal(t, []string{`{"id":"Alloc"}`, `{"id":"Alloc"}`}, bodies)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type temporaryNetError struct{}

func (temporaryNetError) Error() string {
	return "temporary network error"
}

func (temporaryNetError) Timeout() bool {
	return false
}

func (temporaryNetError) Temporary() bool {
	return true
}
