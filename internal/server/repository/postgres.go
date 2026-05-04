package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgerrcode"
	"j30att/observer/internal/retry"
	"j30att/observer/internal/server/model"
)

type PostgresMetricsRepository struct {
	db          *sql.DB
	retryDelays []time.Duration
}

func NewPostgresMetricsRepository(db *sql.DB) *PostgresMetricsRepository {
	return &PostgresMetricsRepository{
		db:          db,
		retryDelays: retry.DefaultDelays,
	}
}

func (r *PostgresMetricsRepository) SaveGauge(name string, value float64) error {
	ctx := context.Background()
	err := retry.Do(ctx, r.retryDelays, isRetriablePostgresError, func() error {
		_, execErr := r.db.ExecContext(
			ctx,
			`
				INSERT INTO metrics (id, type, value, delta)
				VALUES ($1, $2, $3, NULL)
				ON CONFLICT (id, type)
				DO UPDATE SET value = EXCLUDED.value, delta = NULL
			`,
			name,
			model.Gauge,
			value,
		)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("save gauge: %w", err)
	}

	return nil
}

func (r *PostgresMetricsRepository) SaveCounter(name string, delta int64) error {
	ctx := context.Background()
	err := retry.Do(ctx, r.retryDelays, isRetriablePostgresError, func() error {
		_, execErr := r.db.ExecContext(
			ctx,
			`
				INSERT INTO metrics (id, type, delta, value)
				VALUES ($1, $2, $3, NULL)
				ON CONFLICT (id, type)
				DO UPDATE SET delta = metrics.delta + EXCLUDED.delta, value = NULL
			`,
			name,
			model.Counter,
			delta,
		)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("save counter: %w", err)
	}

	return nil
}

func (r *PostgresMetricsRepository) SaveBatch(metrics []model.Metrics) error {
	metrics, err := compactMetrics(metrics)
	if err != nil {
		return err
	}
	if len(metrics) == 0 {
		return nil
	}

	values := make([]string, 0, len(metrics))
	args := make([]any, 0, len(metrics)*4)
	for i, metric := range metrics {
		values = append(values, fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4))
		switch metric.MType {
		case model.Gauge:
			args = append(args, metric.ID, model.Gauge, nil, *metric.Value)
		case model.Counter:
			args = append(args, metric.ID, model.Counter, *metric.Delta, nil)
		}
	}

	ctx := context.Background()
	err = retry.Do(ctx, r.retryDelays, isRetriablePostgresError, func() error {
		_, execErr := r.db.ExecContext(
			ctx,
			`
				INSERT INTO metrics (id, type, delta, value)
				VALUES `+strings.Join(values, ",")+`
				ON CONFLICT (id, type)
				DO UPDATE SET
					delta = CASE
						WHEN EXCLUDED.type = 'counter' THEN metrics.delta + EXCLUDED.delta
						ELSE NULL
					END,
					value = CASE
						WHEN EXCLUDED.type = 'gauge' THEN EXCLUDED.value
						ELSE NULL
					END
			`,
			args...,
		)
		return execErr
	})
	if err != nil {
		return fmt.Errorf("save batch metrics: %w", err)
	}

	return nil
}

func compactMetrics(metrics []model.Metrics) ([]model.Metrics, error) {
	type metricKey struct {
		id    string
		mType string
	}

	compacted := make(map[metricKey]model.Metrics, len(metrics))
	order := make([]metricKey, 0, len(metrics))
	for _, metric := range metrics {
		key := metricKey{id: metric.ID, mType: metric.MType}
		switch metric.MType {
		case model.Gauge:
			if metric.ID == "" || metric.Value == nil {
				return nil, ErrInvalidMetric
			}
		case model.Counter:
			if metric.ID == "" || metric.Delta == nil {
				return nil, ErrInvalidMetric
			}
		default:
			return nil, ErrInvalidMetric
		}

		current, ok := compacted[key]
		if !ok {
			order = append(order, key)
			compacted[key] = metric
			continue
		}

		switch metric.MType {
		case model.Gauge:
			compacted[key] = metric
		case model.Counter:
			delta := *current.Delta + *metric.Delta
			current.Delta = &delta
			compacted[key] = current
		}
	}

	result := make([]model.Metrics, 0, len(compacted))
	for _, key := range order {
		result = append(result, compacted[key])
	}

	return result, nil
}

func (r *PostgresMetricsRepository) Load(metricType, name string) (model.Metrics, error) {
	var metric model.Metrics
	var delta sql.NullInt64
	var value sql.NullFloat64

	ctx := context.Background()
	err := retry.Do(ctx, r.retryDelays, isRetriablePostgresError, func() error {
		return r.db.QueryRowContext(
			ctx,
			`
				SELECT id, type, delta, value
				FROM metrics
				WHERE id = $1 AND type = $2
			`,
			name,
			metricType,
		).Scan(&metric.ID, &metric.MType, &delta, &value)
	})
	if errors.Is(err, sql.ErrNoRows) {
		return model.Metrics{}, ErrMetricNotFound
	}
	if err != nil {
		return model.Metrics{}, fmt.Errorf("load metric: %w", err)
	}

	if delta.Valid {
		metric.Delta = &delta.Int64
	}
	if value.Valid {
		metric.Value = &value.Float64
	}

	return metric, nil
}

func (r *PostgresMetricsRepository) List() []model.Metrics {
	var rows *sql.Rows
	ctx := context.Background()
	err := retry.Do(ctx, r.retryDelays, isRetriablePostgresError, func() error {
		var queryErr error
		rows, queryErr = r.db.QueryContext(
			ctx,
			`
				SELECT id, type, delta, value
				FROM metrics
				ORDER BY id
			`,
		)
		return queryErr
	})
	if err != nil {
		return nil
	}
	defer rows.Close()

	metrics := make([]model.Metrics, 0)
	for rows.Next() {
		var metric model.Metrics
		var delta sql.NullInt64
		var value sql.NullFloat64

		if err := rows.Scan(&metric.ID, &metric.MType, &delta, &value); err != nil {
			return nil
		}

		if delta.Valid {
			metric.Delta = &delta.Int64
		}
		if value.Valid {
			metric.Value = &value.Float64
		}

		metrics = append(metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return nil
	}

	return metrics
}

type sqlStateError interface {
	SQLState() string
}

func isRetriablePostgresError(err error) bool {
	if errors.Is(err, driver.ErrBadConn) {
		return true
	}

	var sqlStateErr sqlStateError
	if !errors.As(err, &sqlStateErr) {
		return false
	}

	return pgerrcode.IsConnectionException(sqlStateErr.SQLState())
}
