package commands

import (
	"errors"
	"strconv"

	"j30att/observer/internal/model"
	"j30att/observer/internal/repository"
)

var ErrUnsupportedMetricType = errors.New("unsupported metric type")

type UpdateMetricCommand struct {
	repo repository.MetricsRepository
}

func NewUpdateMetricCommand(repo repository.MetricsRepository) *UpdateMetricCommand {
	return &UpdateMetricCommand{repo: repo}
}

func (c *UpdateMetricCommand) Execute(metricType, name, rawValue string) error {
	switch metricType {
	case model.Gauge:
		value, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			return err
		}

		return c.repo.SaveGauge(name, value)
	case model.Counter:
		delta, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil {
			return err
		}

		return c.repo.SaveCounter(name, delta)
	default:
		return ErrUnsupportedMetricType
	}
}
