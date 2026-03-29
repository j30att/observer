package controller

import (
	"errors"
	"net/http"
	"strings"

	"j30att/observer/internal/commands"
)

type MetricController struct {
	updateMetricCommand *commands.UpdateMetricCommand
}

func NewMetricController(updateMetricCommand *commands.UpdateMetricCommand) *MetricController {
	return &MetricController{
		updateMetricCommand: updateMetricCommand,
	}
}

func (c *MetricController) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	if contentType := r.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		http.Error(w, "content type must be text/plain", http.StatusBadRequest)
		return
	}

	metricType := r.PathValue("type")
	name := r.PathValue("name")
	value := r.PathValue("value")
	if metricType == "" || name == "" || value == "" {
		http.Error(w, "metric type, name, and value are required", http.StatusNotFound)
		return
	}

	err := c.updateMetricCommand.Execute(metricType, name, value)
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	status := http.StatusBadRequest
	if errors.Is(err, commands.ErrUnsupportedMetricType) {
		status = http.StatusBadRequest
	}

	http.Error(w, err.Error(), status)
}
