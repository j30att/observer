package repository_test

import (
	"database/sql"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

func TestPostgresMetricsRepository(t *testing.T) {
	var (
		db   *sql.DB
		mock sqlmock.Sqlmock
		repo *repository.PostgresMetricsRepository
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

		repo = repository.NewPostgresMetricsRepository(db)
	}

	t.Run("Тест метода SaveGauge", func(t *testing.T) {
		t.Run("Должен сохранить gauge", func(t *testing.T) {
			setup(t)

			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics (id, type, value, delta)")).
				WithArgs("Alloc", model.Gauge, 12.5).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.SaveGauge("Alloc", 12.5)

			require.NoError(t, err)
		})

		t.Run("Должен вернуть ошибку если запрос не выполнен", func(t *testing.T) {
			setup(t)

			saveErr := errors.New("insert failed")
			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics (id, type, value, delta)")).
				WithArgs("Alloc", model.Gauge, 12.5).
				WillReturnError(saveErr)

			err := repo.SaveGauge("Alloc", 12.5)

			require.Error(t, err)
			assert.ErrorIs(t, err, saveErr)
		})
	})

	t.Run("Тест метода SaveCounter", func(t *testing.T) {
		t.Run("Должен сохранить counter", func(t *testing.T) {
			setup(t)

			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics (id, type, delta, value)")).
				WithArgs("PollCount", model.Counter, int64(3)).
				WillReturnResult(sqlmock.NewResult(0, 1))

			err := repo.SaveCounter("PollCount", 3)

			require.NoError(t, err)
		})

		t.Run("Должен вернуть ошибку если запрос не выполнен", func(t *testing.T) {
			setup(t)

			saveErr := errors.New("insert failed")
			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO metrics (id, type, delta, value)")).
				WithArgs("PollCount", model.Counter, int64(3)).
				WillReturnError(saveErr)

			err := repo.SaveCounter("PollCount", 3)

			require.Error(t, err)
			assert.ErrorIs(t, err, saveErr)
		})
	})

	t.Run("Тест метода Load", func(t *testing.T) {
		t.Run("Должен загрузить gauge", func(t *testing.T) {
			setup(t)

			rows := sqlmock.NewRows([]string{"id", "type", "delta", "value"}).
				AddRow("Alloc", model.Gauge, nil, 12.5)
			mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, delta, value")).
				WithArgs("Alloc", model.Gauge).
				WillReturnRows(rows)

			metric, err := repo.Load(model.Gauge, "Alloc")

			require.NoError(t, err)
			assert.Equal(t, "Alloc", metric.ID)
			assert.Equal(t, model.Gauge, metric.MType)
			assert.Nil(t, metric.Delta)
			require.NotNil(t, metric.Value)
			assert.Equal(t, 12.5, *metric.Value)
		})

		t.Run("Должен загрузить counter", func(t *testing.T) {
			setup(t)

			rows := sqlmock.NewRows([]string{"id", "type", "delta", "value"}).
				AddRow("PollCount", model.Counter, int64(3), nil)
			mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, delta, value")).
				WithArgs("PollCount", model.Counter).
				WillReturnRows(rows)

			metric, err := repo.Load(model.Counter, "PollCount")

			require.NoError(t, err)
			assert.Equal(t, "PollCount", metric.ID)
			assert.Equal(t, model.Counter, metric.MType)
			require.NotNil(t, metric.Delta)
			assert.Equal(t, int64(3), *metric.Delta)
			assert.Nil(t, metric.Value)
		})

		t.Run("Должен вернуть not found если метрика не найдена", func(t *testing.T) {
			setup(t)

			mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, delta, value")).
				WithArgs("Alloc", model.Gauge).
				WillReturnError(sql.ErrNoRows)

			_, err := repo.Load(model.Gauge, "Alloc")

			require.ErrorIs(t, err, repository.ErrMetricNotFound)
		})

		t.Run("Должен вернуть ошибку если запрос не выполнен", func(t *testing.T) {
			setup(t)

			loadErr := errors.New("select failed")
			mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, delta, value")).
				WithArgs("Alloc", model.Gauge).
				WillReturnError(loadErr)

			_, err := repo.Load(model.Gauge, "Alloc")

			require.Error(t, err)
			assert.ErrorIs(t, err, loadErr)
		})
	})

	t.Run("Тест метода List", func(t *testing.T) {
		t.Run("Должен вернуть список метрик", func(t *testing.T) {
			setup(t)

			rows := sqlmock.NewRows([]string{"id", "type", "delta", "value"}).
				AddRow("Alloc", model.Gauge, nil, 12.5).
				AddRow("PollCount", model.Counter, int64(3), nil)
			mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, delta, value")).
				WillReturnRows(rows)

			metrics := repo.List()

			require.Len(t, metrics, 2)
			assert.Equal(t, "Alloc", metrics[0].ID)
			assert.Equal(t, model.Gauge, metrics[0].MType)
			require.NotNil(t, metrics[0].Value)
			assert.Equal(t, 12.5, *metrics[0].Value)
			assert.Equal(t, "PollCount", metrics[1].ID)
			assert.Equal(t, model.Counter, metrics[1].MType)
			require.NotNil(t, metrics[1].Delta)
			assert.Equal(t, int64(3), *metrics[1].Delta)
		})

		t.Run("Должен вернуть пустой список если запрос не выполнен", func(t *testing.T) {
			setup(t)

			mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, delta, value")).
				WillReturnError(errors.New("select failed"))

			metrics := repo.List()

			assert.Empty(t, metrics)
		})

		t.Run("Должен вернуть пустой список если строки не сканируются", func(t *testing.T) {
			setup(t)

			rows := sqlmock.NewRows([]string{"id", "type", "delta", "value"}).
				AddRow("Alloc", model.Gauge, "not-int", 12.5)
			mock.ExpectQuery(regexp.QuoteMeta("SELECT id, type, delta, value")).
				WillReturnRows(rows)

			metrics := repo.List()

			assert.Empty(t, metrics)
		})
	})
}
