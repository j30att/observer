package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"j30att/observer/internal/server/model"
)

type PostgresMetricsRepository struct {
	db *sql.DB
}

func NewPostgresMetricsRepository(db *sql.DB) *PostgresMetricsRepository {
	return &PostgresMetricsRepository{db: db}
}

func (r *PostgresMetricsRepository) SaveGauge(name string, value float64) error {
	_, err := r.db.ExecContext(
		context.Background(),
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
	if err != nil {
		return fmt.Errorf("save gauge: %w", err)
	}

	return nil
}

func (r *PostgresMetricsRepository) SaveCounter(name string, delta int64) error {
	_, err := r.db.ExecContext(
		context.Background(),
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
	if err != nil {
		return fmt.Errorf("save counter: %w", err)
	}

	return nil
}

func (r *PostgresMetricsRepository) Load(metricType, name string) (model.Metrics, error) {
	var metric model.Metrics
	var delta sql.NullInt64
	var value sql.NullFloat64

	err := r.db.QueryRowContext(
		context.Background(),
		`
			SELECT id, type, delta, value
			FROM metrics
			WHERE id = $1 AND type = $2
		`,
		name,
		metricType,
	).Scan(&metric.ID, &metric.MType, &delta, &value)
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
	rows, err := r.db.QueryContext(
		context.Background(),
		`
			SELECT id, type, delta, value
			FROM metrics
			ORDER BY id
		`,
	)
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
