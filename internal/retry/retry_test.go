package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	t.Run("Должен ограничить количество дополнительных попыток тремя", func(t *testing.T) {
		attempts := 0
		operationErr := errors.New("temporary error")

		err := Do(context.Background(), []time.Duration{0, 0, 0}, func(error) bool {
			return true
		}, func() error {
			attempts++
			return operationErr
		})

		require.ErrorIs(t, err, operationErr)
		assert.Equal(t, 4, attempts)
	})

	t.Run("Должен остановиться на успешной попытке", func(t *testing.T) {
		attempts := 0
		operationErr := errors.New("temporary error")

		err := Do(context.Background(), []time.Duration{0, 0, 0}, func(error) bool {
			return true
		}, func() error {
			attempts++
			if attempts == 3 {
				return nil
			}
			return operationErr
		})

		require.NoError(t, err)
		assert.Equal(t, 3, attempts)
	})
}
