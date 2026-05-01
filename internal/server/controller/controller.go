package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/model"
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
	if contentType := r.Header.Get("Content-Type"); contentType != "" && !strings.HasPrefix(contentType, "text/plain") {
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
	http.Error(w, err.Error(), status)
}

func (c *MetricController) UpdateMetricJSON(w http.ResponseWriter, r *http.Request) {
	if err := validateJSONContentType(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	metric, err := decodeMetric(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	rawValue, err := rawMetricValue(metric)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.updateMetricCommand.Execute(metric.MType, metric.ID, rawValue); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	savedMetric, err := c.getMetricQuery.Execute(metric.MType, metric.ID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, get.ErrMetricNotFound) {
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	writeJSON(w, http.StatusOK, savedMetric)
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
		if errors.Is(err, get.ErrMetricNotFound) {
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(metricValue(metric)))
}

func (c *MetricController) GetMetricJSON(w http.ResponseWriter, r *http.Request) {
	if err := validateJSONContentType(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	metricRequest, err := decodeMetric(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	metric, err := c.getMetricQuery.Execute(metricRequest.MType, metricRequest.ID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, get.ErrMetricNotFound) {
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	writeJSON(w, http.StatusOK, metric)
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
		return strconv.FormatFloat(*metric.Value, 'g', -1, 64)
	}

	if metric.Delta != nil {
		return strconv.FormatInt(*metric.Delta, 10)
	}

	return ""
}

func validateJSONContentType(r *http.Request) error {
	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		return errors.New("content type is required")
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return fmt.Errorf("parse content type %q: %w", contentType, err)
	}

	if mediaType != "application/json" {
		return fmt.Errorf("content type must be application/json, got %s", mediaType)
	}

	return nil
}

func decodeMetric(body io.Reader) (model.Metrics, error) {
	var metric model.Metrics
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&metric); err != nil {
		return model.Metrics{}, err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return model.Metrics{}, errors.New("request body must contain a single JSON object")
	}

	if metric.ID == "" || metric.MType == "" {
		return model.Metrics{}, errors.New("metric id and type are required")
	}

	return metric, nil
}

func rawMetricValue(metric model.Metrics) (string, error) {
	switch metric.MType {
	case model.Gauge:
		if metric.Value == nil {
			return "", errors.New("gauge value is required")
		}

		return strconv.FormatFloat(*metric.Value, 'f', -1, 64), nil
	case model.Counter:
		if metric.Delta == nil {
			return "", errors.New("counter delta is required")
		}

		return strconv.FormatInt(*metric.Delta, 10), nil
	default:
		return "", update.ErrUnsupportedMetricType
	}
}

func writeJSON(w http.ResponseWriter, status int, metric model.Metrics) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(metric)
}
