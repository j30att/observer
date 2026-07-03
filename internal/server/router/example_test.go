package router_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/rs/zerolog"
	"j30att/observer/internal/server/controller"
	"j30att/observer/internal/server/handlers/get"
	"j30att/observer/internal/server/handlers/getlist"
	"j30att/observer/internal/server/handlers/update"
	"j30att/observer/internal/server/repository"
	"j30att/observer/internal/server/router"
)

func ExampleNewRouter_jsonEndpoints() {
	handler := newExampleRouter()

	updateOneRequest := httptest.NewRequest(
		http.MethodPost,
		"/update",
		strings.NewReader(`{"id":"Alloc","type":"gauge","value":12.5}`),
	)
	updateOneRequest.Header.Set("Content-Type", "application/json")
	updateOneResponse := httptest.NewRecorder()
	handler.ServeHTTP(updateOneResponse, updateOneRequest)

	updateRequest := httptest.NewRequest(
		http.MethodPost,
		"/updates",
		strings.NewReader(`[{"id":"HeapAlloc","type":"gauge","value":42.5},{"id":"PollCount","type":"counter","delta":2}]`),
	)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateResponse := httptest.NewRecorder()
	handler.ServeHTTP(updateResponse, updateRequest)

	valueRequest := httptest.NewRequest(
		http.MethodPost,
		"/value",
		strings.NewReader(`{"id":"Alloc","type":"gauge"}`),
	)
	valueRequest.Header.Set("Content-Type", "application/json")
	valueResponse := httptest.NewRecorder()
	handler.ServeHTTP(valueResponse, valueRequest)

	fmt.Println("single update status:", updateOneResponse.Code)
	fmt.Print(updateOneResponse.Body.String())
	fmt.Println("batch update status:", updateResponse.Code)
	fmt.Println("value status:", valueResponse.Code)
	fmt.Print(valueResponse.Body.String())

	// Output:
	// single update status: 200
	// {"id":"Alloc","type":"gauge","value":12.5}
	// batch update status: 200
	// value status: 200
	// {"id":"Alloc","type":"gauge","value":12.5}
}

func ExampleNewRouter_plainTextEndpoints() {
	handler := newExampleRouter()

	updateRequest := httptest.NewRequest(http.MethodPost, "/update/counter/PollCount/2", nil)
	updateRequest.Header.Set("Content-Type", "text/plain")
	updateResponse := httptest.NewRecorder()
	handler.ServeHTTP(updateResponse, updateRequest)

	valueRequest := httptest.NewRequest(http.MethodGet, "/value/counter/PollCount", nil)
	valueResponse := httptest.NewRecorder()
	handler.ServeHTTP(valueResponse, valueRequest)

	body, _ := io.ReadAll(valueResponse.Body)
	fmt.Println("update status:", updateResponse.Code)
	fmt.Println("value status:", valueResponse.Code)
	fmt.Println("value:", string(body))

	// Output:
	// update status: 200
	// value status: 200
	// value: 2
}

func newExampleRouter() http.Handler {
	repo := repository.NewMetricsRepository()
	updateMetricCommand := update.New(repo)
	getMetricQuery := get.New(repo)
	listMetricsQuery := getlist.New(repo)
	metricController := controller.NewMetricController(updateMetricCommand, getMetricQuery, listMetricsQuery)

	return router.NewRouter(metricController, zerolog.Nop(), nil)
}
