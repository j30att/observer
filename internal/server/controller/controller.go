package controller

import (
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/model"
	"j30att/observer/internal/server/repository"
)

type MetricController struct {
	updateMetricCommand *update.Handler
	getMetricQuery      *get.Handler
	listMetricsQuery    *getlist.Handler
}

func NewMetricController(
	updateMetricCommand *update.Handler,
	getMetricQuery *get.Handler,
	listMetricsQuery *getlist.Handler,
) *MetricController {
	return &MetricController{
		updateMetricCommand: updateMetricCommand,
		getMetricQuery:      getMetricQuery,
		listMetricsQuery:    listMetricsQuery,
	}
}

func (c *MetricController) UpdateMetric(w http.ResponseWriter, r *http.Request) {
	if contentType := r.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		http.Error(w, "content type must be text/plain", http.StatusBadRequest)
		return
	}

	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	value := chi.URLParam(r, "value")
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
	if errors.Is(err, update.ErrUnsupportedMetricType) {
		status = http.StatusBadRequest
	}

	http.Error(w, err.Error(), status)
}

func (c *MetricController) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	if metricType == "" || name == "" {
		http.Error(w, "metric type and name are required", http.StatusNotFound)
		return
	}

	metric, err := c.getMetricQuery.Execute(metricType, name)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, repository.ErrMetricNotFound) {
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(metricValue(metric)))
}

func (c *MetricController) ListMetrics(w http.ResponseWriter, _ *http.Request) {
	metrics := c.listMetricsQuery.Execute()

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	_, _ = fmt.Fprint(w, "<html><body><ul>")
	for _, metric := range metrics {
		_, _ = fmt.Fprintf(
			w,
			"<li>%s: %s</li>",
			html.EscapeString(metric.ID),
			html.EscapeString(metricValue(metric)),
		)
	}
	_, _ = fmt.Fprint(w, "</ul></body></html>")
}

func metricValue(metric model.Metrics) string {
	if metric.Value != nil {
		return strconvFormatFloat(*metric.Value)
	}

	if metric.Delta != nil {
		return strconvFormatInt(*metric.Delta)
	}

	return ""
}

func strconvFormatFloat(value float64) string {
	return fmt.Sprintf("%g", value)
}

func strconvFormatInt(value int64) string {
	return fmt.Sprintf("%d", value)
}
