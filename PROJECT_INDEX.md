# Project Index

Короткая карта проекта для навигации по коду.

## Архитектура

Текущая цепочка зависимостей сервера:

`main -> router -> controller -> handler -> repository`

По слоям:
- `main` собирает зависимости приложения
- `router` связывает маршруты с методами контроллера
- `controller` принимает HTTP-запрос и формирует HTTP-ответ
- `handler` выполняет бизнес-логику
- `repository` сохраняет и читает данные

## Точки входа

### `cmd/server`

- `cmd/server/main.go`  
  Точка входа HTTP-сервера. Создаёт конфиг, репозиторий, команду, контроллер и роутер, затем запускает `http.ListenAndServe`.

### `cmd/agent`

- `cmd/agent/main.go`  
  Заготовка под отдельный агентский процесс. Пока без логики.

## Internal

### `internal/server/router`

- `internal/server/router/router.go`  
  Настройка HTTP-маршрутов. Сейчас регистрирует маршрут:
  - `POST /update/{type}/{name}/{value}` -> `MetricController.UpdateMetric`

### `internal/server/controller`

- `internal/server/controller/controller.go`  
  HTTP-контроллер для метрик.

Основная ответственность:
- проверить `Content-Type`
- достать параметры из URL
- вызвать команду
- вернуть HTTP-статус и текст ошибки при необходимости

Основные элементы:
- `type MetricController struct`
- `func NewMetricController(...)`
- `func (c *MetricController) UpdateMetric(...)`

### `internal/server/handler`

- `internal/server/handler/handler.go`  
  Обработчик обновления метрики.

Основные элементы:
- `type UpdateMetricHandler struct`
- `func NewUpdateMetricHandler(...)`
- `func (c *UpdateMetricHandler) Execute(...)`

Что делает `Execute(...)`:
- для `gauge` парсит `float64` и сохраняет значение
- для `counter` парсит `int64` и увеличивает счётчик
- для неизвестного типа возвращает `ErrUnsupportedMetricType`

### `internal/server/repository`

- `internal/server/repository/repository.go`  
  In-memory репозиторий метрик.

Что хранит:
- `gauges map[string]float64`
- `counters map[string]int64`

Основные элементы:
- `type MetricsRepository interface`
- `type InMemoryMetricsRepository struct`
- `func NewMetricsRepository()`
- `func (r *InMemoryMetricsRepository) SaveGauge(...)`
- `func (r *InMemoryMetricsRepository) SaveCounter(...)`
- `func (r *InMemoryMetricsRepository) Load(...)`

### `internal/model`

- `internal/model/metrics.go`  
  Доменная модель метрики.

Основные элементы:
- `const Gauge`
- `const Counter`
- `type Metrics struct`

### `internal/config`

- `internal/config/server.go`  
  Конфиг HTTP-сервера.

Основные элементы:
- `type ServerConfig struct`
- `func NewServerConfig()`

## Тесты

### `internal/server/controller/controller_test.go`

Покрывает HTTP-поведение контроллера:
- успешный запрос
- неправильный `Content-Type`
- неподдерживаемый тип метрики
- невалидное значение

### `internal/server/handler/handler_test.go`

Покрывает бизнес-логику команды:
- сохранение `gauge`
- накопление `counter`
- ошибка на неподдерживаемый тип
- ошибка на невалидное число

### `internal/server/repository/repository_test.go`

Покрывает репозиторий:
- сохранение и чтение `gauge`
- накопление `counter`
- ошибка `ErrMetricNotFound`

## Примечания

- В корне проекта нет `.go` файлов, поэтому для запуска всех тестов нужно использовать `go test ./...`, а не просто `go test`.
- Основная серверная логика сейчас сосредоточена в `internal/server/controller`, `internal/server/handler` и `internal/server/repository`.
