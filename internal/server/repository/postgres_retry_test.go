package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgerrcode"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
)

type postgresTestError struct {
	code string
}

func (e postgresTestError) Error() string {
	return "postgres error " + e.code
}

func (e postgresTestError) SQLState() string {
	return e.code
}

func TestPostgresMetricsRepositoryRetry(t *testing.T) {
	var (
		db   *sql.DB
		mock sqlmock.Sqlmock
		repo *PostgresMetricsRepository
	)

	setup := func(t *testing.T) {
		t.Helper()

		var err error
		db, mock, err = sqlmock.New()
		require.NoError(t, err)
		t.Cleanup(func() {
			require.NoError(t, mock.ExpectationsWereMet())
			_ = db.Close()
		})

		repo = NewPostgresMetricsRepository(db)
		repo.retryDelays = []time.Duration{0, 0, 0}
	}

	t.Run("Должен повторить запрос при ошибке PostgreSQL Class 08", func(t *testing.T) {
		setup(t)

		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics (id, type, value, delta)")).
			WithArgs("Alloc", model.Gauge, 12.5).
			WillReturnError(postgresTestError{code: pgerrcode.ConnectionFailure})
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics (id, type, value, delta)")).
			WithArgs("Alloc", model.Gauge, 12.5).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.SaveGauge(context.Background(), "Alloc", 12.5)

		require.NoError(t, err)
	})

	t.Run("Не должен повторять запрос при неретраибельной ошибке PostgreSQL", func(t *testing.T) {
		setup(t)

		saveErr := postgresTestError{code: pgerrcode.UniqueViolation}
		mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics (id, type, value, delta)")).
			WithArgs("Alloc", model.Gauge, 12.5).
			WillReturnError(saveErr)

		err := repo.SaveGauge(context.Background(), "Alloc", 12.5)

		require.ErrorIs(t, err, saveErr)
	})
}
