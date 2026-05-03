package ping_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"j30att/observer/internal/server/handlers/ping"
	pingmocks "j30att/observer/internal/server/handlers/ping/mocks"
)

func TestPingHandler(t *testing.T) {
	var (
		handler *ping.Handler
		pinger  *pingmocks.MockPinger
	)

	setup := func(t *testing.T) {
		t.Helper()

		pinger = pingmocks.NewMockPinger(t)
		handler = &[]ping.Handler{ping.New(pinger)}[0]
	}

	t.Run("Тест метода Ping", func(t *testing.T) {
		t.Run("Должен вернуть OK если база доступна", func(t *testing.T) {
			setup(t)

			pinger.EXPECT().PingContext(mock.AnythingOfType("*context.timerCtx")).Return(nil)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ping", nil)
			res := httptest.NewRecorder()

			handler.Ping(res, req)

			assert.Equal(t, http.StatusOK, res.Code)
		})

		t.Run("Должен вернуть internal server error если база недоступна", func(t *testing.T) {
			setup(t)

			pinger.EXPECT().PingContext(mock.AnythingOfType("*context.timerCtx")).
				Return(errors.New("database is unavailable"))
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ping", nil)
			res := httptest.NewRecorder()

			handler.Ping(res, req)

			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})

		t.Run("Должен вернуть internal server error если база не задана", func(t *testing.T) {
			handler := ping.New(nil)
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/ping", nil)
			res := httptest.NewRecorder()

			handler.Ping(res, req)

			assert.Equal(t, http.StatusInternalServerError, res.Code)
		})
	})
}
