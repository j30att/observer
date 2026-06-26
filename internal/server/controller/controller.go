package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"j30att/observer/internal/server/audit"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/model"
)

type MetricController struct {
	updateMetricCommand *update.Handler
	getMetricQuery      *get.Handler
	listMetricsQuery    *getlist.Handler
	auditor             *audit.Subject
}

func NewMetricController(
	updateMetricCommand *update.Handler,
	getMetricQuery *get.Handler,
	listMetricsQuery *getlist.Handler,
	auditors ...*audit.Subject,
) *MetricController {
	var auditor *audit.Subject
	if len(auditors) > 0 {
		auditor = auditors[0]
	}

	return &MetricController{
		updateMetricCommand: updateMetricCommand,
		getMetricQuery:      getMetricQuery,
		listMetricsQuery:    listMetricsQuery,
		auditor:             auditor,
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

	err := c.updateMetricCommand.Execute(r.Context(), metricType, name, value)
	if err == nil {
		c.auditRequest(r, []string{name})
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

	if err := c.updateMetricCommand.Execute(r.Context(), metric.MType, metric.ID, rawValue); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	savedMetric, err := c.getMetricQuery.Execute(r.Context(), metric.MType, metric.ID)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, get.ErrMetricNotFound) {
			status = http.StatusNotFound
		}

		http.Error(w, err.Error(), status)
		return
	}

	c.auditRequest(r, []string{metric.ID})
	writeJSON(w, http.StatusOK, savedMetric)
}

func (c *MetricController) UpdateMetricsJSON(w http.ResponseWriter, r *http.Request) {
	if err := validateJSONContentType(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer func() {
		_ = r.Body.Close()
	}()

	metrics, err := decodeMetrics(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := c.updateMetricCommand.ExecuteBatch(r.Context(), metrics); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	c.auditRequest(r, metricNames(metrics))
	writeJSON(w, http.StatusOK, metrics)
}

func (c *MetricController) GetMetric(w http.ResponseWriter, r *http.Request) {
	metricType := chi.URLParam(r, "type")
	name := chi.URLParam(r, "name")
	if metricType == "" || name == "" {
		http.Error(w, "metric type and name are required", http.StatusNotFound)
		return
	}

	metric, err := c.getMetricQuery.Execute(r.Context(), metricType, name)
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

	metric, err := c.getMetricQuery.Execute(r.Context(), metricRequest.MType, metricRequest.ID)
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

func (c *MetricController) ListMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := c.listMetricsQuery.Execute(r.Context())

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
	metric, err := decodeJSON[model.Metrics](body, "request body must contain a single JSON object")
	if err != nil {
		return model.Metrics{}, err
	}

	if err := validateMetricIdentity(metric); err != nil {
		return model.Metrics{}, errors.New("metric id and type are required")
	}

	return metric, nil
}

func decodeMetrics(body io.Reader) ([]model.Metrics, error) {
	metrics, err := decodeJSON[[]model.Metrics](body, "request body must contain a single JSON array")
	if err != nil {
		return nil, err
	}

	if metrics == nil {
		return nil, errors.New("request body must contain a JSON array")
	}

	for _, metric := range metrics {
		if err := validateMetricIdentity(metric); err != nil {
			return nil, errors.New("metric id and type are required")
		}
	}

	return metrics, nil
}

func decodeJSON[T any](body io.Reader, extraValueError string) (T, error) {
	var payload T
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(&payload); err != nil {
		return payload, err
	}

	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return payload, errors.New(extraValueError)
	}

	return payload, nil
}

func validateMetricIdentity(metric model.Metrics) error {
	if metric.ID == "" || metric.MType == "" {
		return errors.New("metric id and type are required")
	}

	return nil
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

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func (c *MetricController) auditRequest(r *http.Request, metricNames []string) {
	if c.auditor == nil {
		return
	}

	c.auditor.Notify(r.Context(), audit.NewEvent(metricNames, requestIP(r)))
}

func metricNames(metrics []model.Metrics) []string {
	names := make([]string, 0, len(metrics))
	for _, metric := range metrics {
		names = append(names, metric.ID)
	}

	return names
}

func requestIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}
