# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m v2 template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/v2 .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Шифрование сообщений агента

Асимметричное шифрование включается только при заданном пути к ключу:

- агенту через `-crypto-key` или `CRYPTO_KEY` передаётся PEM-файл с публичным RSA-ключом;
- серверу через `-crypto-key` или `CRYPTO_KEY` передаётся PEM-файл с приватным RSA-ключом.

Переменная окружения `CRYPTO_KEY` имеет приоритет над флагом. Если путь не задан, агент и сервер работают без шифрования.

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Профилирование памяти

Добавлены бенчмарки для основных операций, которые часто выполняются во время работы агента и сервера:

- `BenchmarkMetricsRepositorySnapshot`: копирование снимка метрик агента перед отправкой.
- `BenchmarkHTTPSenderSend`: подготовка и отправка batch с gauge/counter метриками.
- `BenchmarkInMemoryMetricsRepositoryList`: сбор списка серверных метрик для ответа и сохранения snapshot.

Базовый профиль памяти сохранён в `profiles/base.pprof`, повторный профиль после оптимизации - в `profiles/result.pprof`.

Команды для воспроизведения:

```shell
go test ./internal/agent/repository ./internal/agent/senders ./internal/server/repository -bench=. -benchmem
go test ./internal/agent/senders -bench=BenchmarkHTTPSenderSend -benchmem -memprofile=profiles/base.pprof
go test ./internal/agent/senders -bench=BenchmarkHTTPSenderSend -benchmem -memprofile=profiles/result.pprof
pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Вывод `pprof -top -diff_base=profiles/base.pprof profiles/result.pprof`:

```text
File: senders.test
Type: alloc_space
Time: 2026-06-28 22:17:35 +08
Showing nodes accounting for -70.27MB, 2.60% of 2700.48MB total
Dropped 9 nodes (cum <= 13.50MB)
      flat  flat%   sum%        cum   cum%
  -56.41MB  2.09%  2.09%   -67.12MB  2.49%  compress/flate.NewWriter (inline)
  -11.59MB  0.43%  2.52%   -69.10MB  2.56%  j30att/observer/internal/agent/senders.(*HTTPSender).Send
  -10.20MB  0.38%  2.49%   -10.20MB  0.38%  compress/flate.(*compressor).initDeflate (inline)
   -6.71MB  0.25%  2.73%    -6.71MB  0.25%  net/http.init.func16
   -4.51MB  0.17%  2.73%    -4.51MB  0.17%  compress/flate.(*huffmanEncoder).generate
   -2.50MB 0.093%  2.83%    -2.50MB 0.093%  compress/flate.newHuffmanEncoder (inline)
         0     0%  2.60%   -57.51MB  2.13%  j30att/observer/internal/agent/senders.(*HTTPSender).sendMetrics
         0     0%  2.60%   -66.84MB  2.47%  j30att/observer/internal/agent/senders.BenchmarkHTTPSenderSend
         0     0%  2.60%   -70.12MB  2.60%  j30att/observer/internal/compression.CompressGzip
```

Отрицательные значения в diff-профиле показывают снижение потребления памяти относительно `profiles/base.pprof`.

Результаты `benchmem` на Apple M1 Pro:

| Бенчмарк | До | После |
| --- | ---: | ---: |
| `BenchmarkMetricsRepositorySnapshot` | `9556 ns/op`, `13904 B/op`, `22 allocs/op` | `4664 ns/op`, `7088 B/op`, `8 allocs/op` |
| `BenchmarkHTTPSenderSend` | `450529 ns/op`, `877057 B/op`, `320 allocs/op` | `356910 ns/op`, `875593 B/op`, `121 allocs/op` |
| `BenchmarkInMemoryMetricsRepositoryList` | `24058 ns/op`, `15304 B/op`, `204 allocs/op` | `22452 ns/op`, `15496 B/op`, `6 allocs/op` |

Что было оптимизировано:

- В `MetricsRepository.Snapshot` карты создаются сразу с нужной capacity.
- В `HTTPSender.Send` значения gauge/counter хранятся в backing slices, поэтому для полей `Value` и `Delta` больше не создаётся отдельная heap-аллокация на каждую метрику.
- В `InMemoryMetricsRepository.List` применён такой же подход с backing slices для значений метрик; количество аллокаций снизилось с 204 до 6 на наборе из 200 метрик.
