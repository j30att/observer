package ping_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/handlers/ping"
)

type fakePinger struct {
	err error
}

func (p fakePinger) PingContext(context.Context) error {
	return p.err
}

func TestPingReturnsOKWhenDatabaseIsAvailable(t *testing.T) {
	handler := ping.New(fakePinger{})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	handler.Ping(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestPingReturnsInternalServerErrorWhenDatabaseIsUnavailable(t *testing.T) {
	handler := ping.New(fakePinger{err: errors.New("database is unavailable")})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	handler.Ping(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestPingReturnsInternalServerErrorWithoutDatabase(t *testing.T) {
	handler := ping.New(nil)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()

	handler.Ping(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}
