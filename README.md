# ruglogger

`ruglogger` — это небольшая библиотека структурированного логирования для Go.
Она построена поверх стандартного пакета `log/slog` и добавляет удобный API,
логирование в файл, Gin middleware, логгеры уровня запроса и ошибки приложения,
которые безопасно отдавать клиенту.

Библиотека подходит для сервисов, где нужны понятные логи для разработчика и
одновременно удобный формат для машинной обработки: `jq`, Loki, ELK, ClickHouse
или собственного сборщика логов.

## Возможности

- Структурированные логи в JSON или текстовом формате.
- Запись в stdout, файл или любой `io.Writer`.
- Глобальный логгер для простых приложений.
- Отдельные экземпляры логгера для сервисов, воркеров, тестов и tenant'ов.
- Дочерние логгеры с наследованием полей: `service`, `request_id`, `tenant_id`.
- Изменение уровня логирования во время работы приложения.
- Логирование с поддержкой `context.Context`.
- Gin middleware для логирования HTTP-запросов.
- Логирование query, headers, cookies и body с маскированием чувствительных данных.
- Ошибки приложения с HTTP-статусом, стабильным app code, публичным сообщением и
  внутренней причиной для логов.
- Helper для WebSocket-ошибок.

## Пакеты

| Пакет | Назначение |
| --- | --- |
| `ruglog` | Основной структурированный логгер и его настройки. |
| `rugerror` | Ошибки приложения для логов и HTTP-ответов. |
| `middleware` | Gin middleware для логирования запросов и обработки ошибок. |

## Установка

```bash
go get github.com/PrototypeSirius/ruglogger
```

Обычно в приложении используются такие импорты:

```go
import (
	"github.com/PrototypeSirius/ruglogger/middleware"
	apperror "github.com/PrototypeSirius/ruglogger/rugerror"
	logger "github.com/PrototypeSirius/ruglogger/ruglog"
)
```

## Быстрый старт

Пример ниже показывает минимальную настройку Gin-сервиса, который пишет
структурированные логи в файл `logs/app.log`.

```go
package main

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/PrototypeSirius/ruglogger/middleware"
	apperror "github.com/PrototypeSirius/ruglogger/rugerror"
	logger "github.com/PrototypeSirius/ruglogger/ruglog"
)

func main() {
	// Логгер создает файл, но не создает родительские папки.
	// Поэтому папку logs нужно создать заранее.
	if err := os.MkdirAll("logs", 0o755); err != nil {
		panic(err)
	}

	log := logger.MustNew(
		// JSON — рекомендуемый формат для production.
		logger.WithFormat(logger.FormatJSON),

		// INFO означает, что TRACE и DEBUG не попадут в лог.
		logger.WithLevel(logger.LevelInfo),

		// Логи будут дописываться в файл logs/app.log.
		logger.WithFile("logs/app.log"),

		// Поле service будет добавлено в каждую строку этого логгера.
		logger.WithField("service", "orders-api"),
	)
	defer log.Close()

	router := gin.New()

	// StructuredLogHandler создает request logger:
	// в него попадут method, path, ip, request_id и другие поля запроса.
	router.Use(middleware.StructuredLogHandler(
		middleware.WithRequestLogger(log),
		middleware.WithHeaderLogging("X-Request-ID"),
		middleware.WithRequestBodyLogging(16*1024),
	))

	// ErrorHandler превращает AppError в JSON-ответ и пишет ошибку в лог.
	router.Use(middleware.ErrorHandler(
		middleware.WithErrorLogger(log),
	))

	router.POST("/orders", func(c *gin.Context) {
		// RequestLogger возвращает логгер, привязанный к текущему HTTP-запросу.
		middleware.RequestLogger(c).Info("creating order", logger.Fields{
			"phase": "validation",
		})

		c.Error(
			apperror.BadRequestError(nil, 1001, "Invalid order data").WithDetails(map[string]any{
				"field": "items",
			}),
		)
	})

	_ = http.ListenAndServe(":8080", router)
}
```

Пример строк в `logs/app.log`:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"creating order","service":"orders-api","method":"POST","path":"/orders","phase":"validation"}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"Invalid order data","service":"orders-api","app_code":1001,"details":{"field":"items"},"status":400}
```

Реальные значения `timestamp`, `ip`, `latency_ms`, `request_id` и других полей
зависят от конкретного запроса.

## Важное замечание про запись в файл

`WithFile("logs/app.log")` добавляет файл к стандартному stdout. Это значит, что
одна и та же строка будет записана и в консоль, и в файл.

Если нужен режим строго только в файл, откройте файл самостоятельно и передайте
его через `WithOutput`.

```go
file, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
if err != nil {
	panic(err)
}
defer file.Close()

log := logger.MustNew(
	// WithOutput заменяет stdout на переданный writer.
	logger.WithOutput(file),
	logger.WithFormat(logger.FormatJSON),
	logger.WithLevel(logger.LevelInfo),
)
```

## Основные понятия

### Уровни логирования

| Уровень | Когда использовать |
| --- | --- |
| `LevelTrace` | Очень подробные технические события. |
| `LevelDebug` | Отладочная информация во время разработки. |
| `LevelInfo` | Обычные события приложения. |
| `LevelWarn` | Что-то пошло неидеально, но приложение продолжает работать. |
| `LevelError` | Операция завершилась ошибкой. |
| `LevelFatal` | Критическая ошибка. После записи вызывается `exitFunc(1)`. |

Пример:

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithFormat(logger.FormatJSON),
	logger.WithLevel(logger.LevelWarn),
)
defer log.Close()

log.Debug("debug skipped", nil)      // Не запишется, потому что уровень WARN.
log.Info("info skipped", nil)        // Не запишется, потому что уровень WARN.
log.Warn("cache is slow", 2001, nil) // Запишется.
log.Error("database failed", errors.New("connection refused"), 3001, nil)
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"WARN","message":"cache is slow","app_code":2001}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"database failed","app_code":3001,"error":"connection refused"}
```

### Поля

Поля — это дополнительные данные в формате ключ-значение. Именно они делают лог
структурированным и удобным для поиска.

```go
log.Info("payment accepted", logger.Fields{
	"order_id": 101,
	"amount":   2500,
	"currency": "RUB",
})
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"payment accepted","order_id":101,"amount":2500,"currency":"RUB"}
```

### Поля по умолчанию

Поля, переданные через `WithField` или `WithFields`, будут добавлены в каждую
строку конкретного логгера.

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithFields(logger.Fields{
		"service":     "billing-api",
		"environment": "production",
	}),
)
defer log.Close()

log.Info("invoice created", logger.Fields{
	"invoice_id": 9001,
})
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"invoice created","service":"billing-api","environment":"production","invoice_id":9001}
```

### Дочерние логгеры

Дочерний логгер наследует настройки родителя и добавляет свои поля. Это удобно
для воркеров, tenant'ов, пользователей, request id и trace id.

```go
baseLog := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithField("service", "billing-api"),
)
defer baseLog.Close()

workerLog := baseLog.WithFields(logger.Fields{
	"component": "email-worker",
	"worker_id": "worker-1",
})

workerLog.Info("job started", logger.Fields{
	"job_id": "job-777",
})
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"job started","service":"billing-api","component":"email-worker","worker_id":"worker-1","job_id":"job-777"}
```

## Настройка логгера

### Рекомендуемый шаблон для сервиса

```go
func NewAppLogger() (*logger.Logger, error) {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		return nil, err
	}

	return logger.New(
		logger.WithFile("logs/app.log"),
		logger.WithFormat(logger.FormatJSON),
		logger.WithLevelString(os.Getenv("LOG_LEVEL")),
		logger.WithField("service", "orders-api"),
		logger.WithField("environment", os.Getenv("APP_ENV")),
	)
}
```

### Опции

| Опция | Что делает |
| --- | --- |
| `WithLevel(level)` | Устанавливает минимальный уровень логирования. |
| `WithLevelString(value)` | Читает `trace`, `debug`, `info`, `warn`, `error`, `fatal`. Пустая строка означает `info`. |
| `WithFormat(format)` | Выбирает `FormatJSON` или `FormatText`. |
| `WithTimeFormat(format)` | Меняет формат поля `timestamp`. |
| `WithAddSource(true)` | Добавляет файл, функцию и номер строки вызова. |
| `WithOutput(writer)` | Пишет в один writer и заменяет stdout. |
| `WithOutputs(writers...)` | Пишет одну строку сразу в несколько writer'ов. |
| `WithFile(path)` | Дописывает логи в файл и оставляет stdout включенным. |
| `WithField(key, value)` | Добавляет одно поле по умолчанию. |
| `WithFields(fields)` | Добавляет несколько полей по умолчанию. |
| `WithReplaceAttr(fn)` | Изменяет или скрывает атрибуты перед записью. |
| `WithHandler(handler)` | Использует собственный `slog.Handler`. |
| `WithExitFunc(fn)` | Заменяет функцию, которую вызывает `Fatal`. Полезно в тестах. |

### JSON и text формат

JSON рекомендуется для production:

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithFormat(logger.FormatJSON),
)
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"service started"}
```

Text удобнее читать глазами:

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithFormat(logger.FormatText),
)
```

В файле:

```text
timestamp=2026-05-11T13:00:00.000000000+03:00 level=INFO message="service started"
```

### Изменение уровня во время работы

```go
log.SetLevel(logger.LevelDebug)

log.Debug("debug logging enabled", logger.Fields{
	"feature": "runtime-level",
})
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"DEBUG","message":"debug logging enabled","feature":"runtime-level"}
```

### Свой формат времени

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithTimeFormat("2006-01-02 15:04:05"),
)
defer log.Close()

log.Info("custom time format", nil)
```

В файле:

```json
{"timestamp":"2026-05-11 13:00:00","level":"INFO","message":"custom time format"}
```

### Source location

`WithAddSource(true)` показывает место вызова. Это удобно при отладке, но может
создавать лишний шум в production.

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithAddSource(true),
)
defer log.Close()

log.Info("with source", nil)
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","source":{"function":"main.main","file":"C:/project/main.go","line":20},"message":"with source"}
```

### Маскирование собственных полей

Используйте `WithReplaceAttr`, если значение нельзя записывать в лог.

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithReplaceAttr(func(groups []string, attr slog.Attr) slog.Attr {
		if attr.Key == "user_email" {
			attr.Value = slog.StringValue("[REDACTED]")
		}
		return attr
	}),
)
defer log.Close()

log.Info("user updated", logger.Fields{
	"user_email": "secret@example.com",
})
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"user updated","user_email":"[REDACTED]"}
```

## Примеры API

### Локальный логгер

Локальный логгер лучше использовать, когда хочется явно контролировать
зависимости и проще писать тесты.

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithField("service", "payments"),
)
defer log.Close()

log.Info("payment service started", nil)
```

### Глобальный логгер

Глобальный логгер удобен для небольших приложений или старого кода, где неудобно
передавать `log` во все функции.

```go
if err := logger.InitWithOptions(
	logger.WithFile("logs/app.log"),
	logger.WithFormat(logger.FormatJSON),
	logger.WithLevel(logger.LevelInfo),
	logger.WithField("service", "billing-api"),
); err != nil {
	panic(err)
}
defer logger.Close()

logger.Info("global logger started", nil)
logger.Warn("global warning", 2001, logger.Fields{"retry": true})
logger.Error("global error", errors.New("boom"), 5001, nil)
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"global logger started","service":"billing-api"}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"WARN","message":"global warning","service":"billing-api","app_code":2001,"retry":true}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"global error","service":"billing-api","app_code":5001,"error":"boom"}
```

### Основные методы

| Метод | Пример |
| --- | --- |
| `Trace(msg, fields)` | `log.Trace("cache lookup", logger.Fields{"key": "user:42"})` |
| `Debug(msg, fields)` | `log.Debug("sql query", logger.Fields{"query": "select ..."})` |
| `Info(msg, fields)` | `log.Info("created", logger.Fields{"id": 1})` |
| `Warn(msg, appCode, fields)` | `log.Warn("slow dependency", 2001, nil)` |
| `Error(msg, err, appCode, fields)` | `log.Error("failed", err, 5001, nil)` |
| `Fatal(msg, err, appCode, fields)` | `log.Fatal("cannot start", err, 9001, nil)` |
| `Log(level, msg, err, appCode, fields)` | `log.Log(logger.LevelInfo, "dynamic", nil, 0, nil)` |
| `WithField(key, value)` | `log.WithField("request_id", "req-123")` |
| `WithFields(fields)` | `log.WithFields(logger.Fields{"tenant_id": "acme"})` |
| `WithError(err)` | `log.WithError(err).Error("failed", err, 5001, nil)` |
| `WithAppCode(code)` | `log.WithAppCode(1001).Info("business event", nil)` |
| `WithGroup(name)` | `log.WithGroup("request").WithField("path", "/orders")` |
| `IntoContext(ctx)` | `ctx = log.IntoContext(ctx)` |
| `Slog()` | `log.Slog().Info("native slog call")` |
| `Close()` | `defer log.Close()` |

Нюанс: метод `log.Debug` принимает `(msg, fields)`, а глобальная функция
`logger.Debug` принимает `(msg, appCode, fields)`.

### Динамический уровень

`Log` полезен, когда уровень вычисляется во время выполнения.

```go
level := logger.LevelInfo
if queueSize > 1000 {
	level = logger.LevelWarn
}

log.Log(level, "queue size checked", nil, 0, logger.Fields{
	"queue_size": queueSize,
})
```

В файле при `queueSize = 1500`:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"WARN","message":"queue size checked","queue_size":1500}
```

### Логгер в context.Context

Логгер можно положить в `context.Context`, чтобы нижний уровень кода мог писать
логи с request-полями.

```go
ctx := log.WithField("request_id", "req-123").IntoContext(context.Background())

func loadUser(ctx context.Context, userID int) {
	requestLog := logger.FromContext(ctx)

	requestLog.InfoContext(ctx, "loading user", logger.Fields{
		"user_id": userID,
	})
}

loadUser(ctx, 42)
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"loading user","request_id":"req-123","user_id":42}
```

### Проверка Enabled

`Enabled` помогает не собирать дорогие debug-данные, если уровень выключен.

```go
if log.Enabled(context.Background(), logger.LevelDebug) {
	log.Debug("expensive debug data", logger.Fields{
		"payload": buildLargeDebugPayload(),
	})
}
```

### Группы полей

```go
requestLog := log.WithGroup("request").WithFields(logger.Fields{
	"method": "POST",
	"path":   "/orders",
})

requestLog.Info("request received", nil)
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"request received","request":{"method":"POST","path":"/orders"}}
```

### MergeFields

`MergeFields` объединяет несколько карт. Если ключ повторяется, побеждает
последнее значение.

```go
baseFields := logger.Fields{"service": "billing-api", "env": "prod"}
requestFields := logger.Fields{"request_id": "req-123", "env": "stage"}

log.Info("merged fields", logger.MergeFields(baseFields, requestFields))
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"merged fields","service":"billing-api","env":"stage","request_id":"req-123"}
```

## Gin request logging

`StructuredLogHandler` пишет лог по каждому HTTP-запросу после завершения
handler'а. Он добавляет базовые поля запроса и может логировать headers, cookies,
query и body.

### Рекомендуемый порядок middleware

```go
router := gin.New()

router.Use(middleware.StructuredLogHandler(
	middleware.WithRequestLogger(log),
	middleware.WithHeaderLogging("X-Request-ID"),
	middleware.WithRequestBodyLogging(16*1024),
))

router.Use(middleware.ErrorHandler(
	middleware.WithErrorLogger(log),
))
```

Порядок важен:

1. `StructuredLogHandler` должен идти первым. Он создает request logger.
2. `ErrorHandler` должен идти после него. Он логирует ошибки и возвращает ответы.

### Логгер внутри handler'а

```go
router.POST("/orders", func(c *gin.Context) {
	middleware.RequestLogger(c).Info("inside handler", logger.Fields{
		"phase": "validation",
	})

	c.JSON(http.StatusCreated, gin.H{"ok": true})
})
```

Пример запроса:

```http
POST /orders?token=secret&search=bag
X-Request-ID: req-123
Content-Type: application/json

{"sku":"bag"}
```

Пример логов:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"inside handler","request_id":"req-123","method":"POST","path":"/orders","phase":"validation"}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"request completed","request_id":"req-123","method":"POST","path":"/orders","status":201,"latency_ms":1,"query":"search=bag&token=%5BREDACTED%5D","headers":{"X-Request-Id":"req-123"},"body":"{\"sku\":\"bag\"}"}
```

### Опции middleware

| Опция | Что делает |
| --- | --- |
| `WithRequestLogger(log)` | Использует ваш логгер вместо глобального. |
| `WithRequestMessages(success, error)` | Меняет сообщения request-логов. |
| `WithRequestSkipper(fn)` | Пропускает логирование, если функция вернула `true`. |
| `WithSkipPaths(paths...)` | Пропускает точные пути, например `/healthz`. |
| `WithQueryLogging()` | Включает логирование query. Включено по умолчанию. |
| `WithoutQueryLogging()` | Выключает логирование query. |
| `WithHeaderLogging(names...)` | Логирует все headers или только перечисленные. |
| `WithCookieLogging(names...)` | Логирует все cookies или только перечисленные. |
| `WithRequestBodyLogging(limit)` | Захватывает body до указанного лимита. |
| `WithBodyFormatter(fn)` | Заменяет raw body на свое значение. |
| `WithRequestExtraFields(fn)` | Добавляет свои поля из `gin.Context`. |
| `WithRequestLevelResolver(fn)` | Выбирает уровень для каждого запроса. |
| `WithRequestIDHeaders(names...)` | Выбирает headers, из которых брать `request_id`. |
| `WithRedactedHeaders(names...)` | Добавляет headers, которые нужно скрывать. |
| `WithRedactedCookies(names...)` | Добавляет cookies, которые нужно скрывать. |
| `WithRedactedQueryParams(names...)` | Добавляет query-параметры, которые нужно скрывать. |

### Маскирование чувствительных данных

Middleware уже скрывает популярные секреты.

Headers:

- `Authorization`
- `Cookie`
- `Set-Cookie`
- `X-API-Key`
- `X-Auth-Token`

Cookies:

- `session`
- `sessionid`
- `session_id`
- `csrf`
- `csrf_token`
- `refresh_token`

Query parameters:

- `password`
- `token`
- `access_token`
- `refresh_token`
- `secret`
- `api_key`

Пример:

```go
router.Use(middleware.StructuredLogHandler(
	middleware.WithRequestLogger(log),
	middleware.WithHeaderLogging("X-Request-ID", "Authorization"),
	middleware.WithCookieLogging("session"),
	middleware.WithRedactedQueryParams("email"),
))
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"request completed","headers":{"Authorization":"[REDACTED]","X-Request-Id":"req-123"},"cookies":{"session":"[REDACTED]"},"query":"email=%5BREDACTED%5D"}
```

### Форматирование body

Используйте `WithBodyFormatter`, если body может содержать приватные данные.

```go
router.Use(middleware.StructuredLogHandler(
	middleware.WithRequestLogger(log),
	middleware.WithRequestBodyLogging(16*1024),
	middleware.WithBodyFormatter(func(c *gin.Context, body []byte) any {
		return logger.Fields{
			"bytes": len(body),
		}
	}),
))
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"request completed","body":{"bytes":13},"status":200}
```

## Ошибки приложения

`rugerror.AppError` отделяет внутреннюю ошибку от публичного HTTP-ответа.

Внутри `AppError` есть:

- внутренний `error` для логов;
- публичное `message` для клиента;
- HTTP-статус;
- стабильный `app_code`;
- необязательные публичные `details`.

### Создание AppError

```go
router.POST("/users", func(c *gin.Context) {
	err := errors.New("sql: duplicate key value violates unique constraint")

	c.Error(
		apperror.ConflictError(err, 40901, "User already exists").WithDetails(map[string]any{
			"field": "email",
		}),
	)
})
```

Ответ клиенту:

```json
{"app_code":40901,"details":{"field":"email"},"message":"User already exists"}
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"User already exists","app_code":40901,"error":"sql: duplicate key value violates unique constraint","details":{"field":"email"},"status":409}
```

### Конструкторы AppError

| Конструктор | HTTP-статус |
| --- | --- |
| `BadRequestError(err, code, message)` | `400 Bad Request` |
| `UnauthorizedError(err, code, message)` | `401 Unauthorized` |
| `ForbiddenError(err, code, message)` | `403 Forbidden` |
| `NotFoundError(err, code, message)` | `404 Not Found` |
| `ConflictError(err, code, message)` | `409 Conflict` |
| `SystemError(err, code, message)` | `500 Internal Server Error` |
| `CustomError(err, status, code, message)` | Любой статус |

Если `message` пустой, большинство конструкторов подставит стандартное
публичное сообщение.

### Необработанные ошибки

Если handler добавил обычную ошибку, а не `AppError`, `ErrorHandler` вернет
безопасный `500` и запишет настоящую ошибку в лог.

```go
router.GET("/boom", func(c *gin.Context) {
	c.Error(errors.New("database is down"))
})
```

Ответ клиенту:

```json
{"app_code":9999,"message":"Internal Server Error"}
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"Unhandled system error","app_code":9999,"error":"database is down","status":500}
```

Fallback-код и сообщение можно изменить:

```go
router.Use(middleware.ErrorHandler(
	middleware.WithErrorLogger(log),
	middleware.WithUnhandledError(9500, "Unexpected server error"),
))
```

## WebSocket-ошибки

В пакете `middleware` есть helper для WebSocket-ошибок.

```go
func onWebSocketError(conn *websocket.Conn, err error) {
	middleware.HandleWebSocketError(conn, err, "websocket operation failed")
}
```

Если ошибка не является `AppError`, она будет превращена в системную ошибку с
кодом `9999`.

Сообщение клиенту по WebSocket:

```json
{"type":"error","message":"WebSocket internal error","app_code":9999}
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"websocket operation failed","app_code":9999,"error":"some websocket error","protocol":"websocket"}
```

## Рекомендации по безопасности

- Для production используйте `FormatJSON`.
- Перед `WithFile` создавайте директорию через `os.MkdirAll`.
- Если используете `WithFile`, вызывайте `Close()`.
- Не пишите в логи пароли, токены, приватные документы и полные body без
  необходимости.
- Для body с чувствительными данными используйте `WithBodyFormatter`.
- Чтобы убрать шум, пропускайте `/healthz` и `/metrics` через `WithSkipPaths`.
- Используйте стабильные `app_code`, чтобы клиентам и поддержке было проще
  находить причину ошибки.

## Troubleshooting

### VS Code не видит `github.com/gorilla/websocket`

Если `middleware/error.go` импортирует `github.com/gorilla/websocket`, модуль
должен быть указан в `go.mod`:

```go
require (
	github.com/gin-gonic/gin v1.11.0
	github.com/gorilla/websocket v1.5.3
)
```

Если зависимость пропала после `go mod tidy`, чаще всего причина одна из этих:

- зависимость удалили из `go.mod` вручную;
- IDE еще не перечитала модуль;
- команда была запущена не из корня проекта;
- файл в Windows оказался `ReparsePoint`/cloud placeholder, и Go tooling
  некорректно его анализирует.

В этом репозитории `middleware/error.go` и `middleware/logger.go` были заменены
обычными файлами без атрибута `ReparsePoint`. После этого `go test ./...`
успешно проходит.

Чтобы обновить окружение в VS Code:

1. Выполните `go mod tidy`.
2. Выполните `go test ./...`.
3. Перезапустите Go language server: `Go: Restart Language Server`.
4. Если подсветка осталась, закройте и откройте папку проекта заново.

### Логи пишутся и в консоль, и в файл

Это ожидаемое поведение при использовании `WithFile`. Если нужен только файл,
используйте `os.OpenFile` вместе с `WithOutput(file)`.

### Body не попадает в лог

Body захватывается во время чтения handler'ом. Если handler не читает body, в
лог может быть нечего записывать. `multipart/form-data` намеренно не логируется.

## Тестирование

```bash
go test ./...
```
