# ruglogger: подробные примеры использования

Этот файл показывает, как пользоваться библиотекой, если логи нужно писать в
отдельный файл, а также что будет появляться внутри этого файла.

В примерах используются такие импорты:

```go
import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/PrototypeSirius/ruglogger/middleware"
	apperror "github.com/PrototypeSirius/ruglogger/rugerror"
	logger "github.com/PrototypeSirius/ruglogger/ruglog"
)
```

> В примерах вывода `timestamp` сокращен до читаемого значения. В реальном
> файле будет текущее время.

## 1. Самый простой вариант: писать в отдельный файл

`WithFile` создает файл, если его еще нет, и дописывает новые строки в конец.
Важно: текущая реализация `WithFile` добавляет файл к стандартному stdout, то
есть лог будет и в консоли, и в файле.

```go
func main() {
	// Папку нужно создать заранее: WithFile создает файл, но не создает папки.
	if err := os.MkdirAll("logs", 0o755); err != nil {
		panic(err)
	}

	log := logger.MustNew(
		// JSON удобен для машинной обработки: grep, jq, ELK, Loki, ClickHouse.
		logger.WithFormat(logger.FormatJSON),

		// Все сообщения ниже INFO будут отфильтрованы.
		logger.WithLevel(logger.LevelInfo),

		// Файл будет создан как logs/app.log.
		logger.WithFile("logs/app.log"),

		// Поле service попадет в каждую строку этого логгера.
		logger.WithField("service", "billing-api"),
	)
	defer log.Close() // Закрывает файл, который открыл WithFile.

	log.Info("service started", logger.Fields{
		"version": "1.0.0",
	})
}
```

В `logs/app.log` появится:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"service started","service":"billing-api","version":"1.0.0"}
```

## 2. Режим строго только в файл, без stdout

Если не нужен вывод в консоль, файл можно открыть самостоятельно и передать
через `WithOutput`. Тогда закрывать файл нужно самому.

```go
func main() {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		panic(err)
	}

	file, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	log := logger.MustNew(
		// WithOutput заменяет стандартный stdout на переданный writer.
		logger.WithOutput(file),
		logger.WithFormat(logger.FormatJSON),
		logger.WithLevel(logger.LevelDebug),
		logger.WithField("service", "billing-api"),
	)

	log.Debug("debug message", logger.Fields{"step": "init"})
	log.Info("service started", nil)
}
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"DEBUG","message":"debug message","service":"billing-api","step":"init"}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"service started","service":"billing-api"}
```

## 3. Текстовый формат вместо JSON

Текстовый формат удобен человеку, но хуже подходит для автоматического поиска
по полям.

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithFormat(logger.FormatText),
	logger.WithLevel(logger.LevelInfo),
	logger.WithField("service", "billing-api"),
)
defer log.Close()

log.Info("payment accepted", logger.Fields{
	"order_id": 101,
	"amount":   2500,
})
```

В файле:

```text
timestamp=2026-05-11T13:00:00.000000000+03:00 level=INFO message="payment accepted" service=billing-api amount=2500 order_id=101
```

## 4. Уровни логирования

Доступные уровни:

- `LevelTrace` - максимально подробные технические события.
- `LevelDebug` - отладочная информация.
- `LevelInfo` - обычные бизнес-события.
- `LevelWarn` - проблема есть, но приложение продолжает работать.
- `LevelError` - ошибка операции.
- `LevelFatal` - критическая ошибка, после записи вызывается `exitFunc(1)`.

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithFormat(logger.FormatJSON),
	logger.WithLevel(logger.LevelWarn), // INFO и DEBUG не попадут в файл.
)
defer log.Close()

log.Debug("debug skipped", nil)      // Не запишется.
log.Info("info skipped", nil)        // Не запишется.
log.Warn("cache is slow", 2001, nil) // Запишется.
log.Error("db failed", errors.New("connection refused"), 3001, nil)
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"WARN","message":"cache is slow","app_code":2001}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"db failed","app_code":3001,"error":"connection refused"}
```

Уровень можно изменить во время работы:

```go
log.SetLevel(logger.LevelDebug) // С этого момента DEBUG начнет писаться.
log.Debug("debug enabled", logger.Fields{"feature": "runtime-level"})
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"DEBUG","message":"debug enabled","feature":"runtime-level"}
```

## 5. Уровень из строки

`WithLevelString` удобно использовать с переменными окружения.

```go
level := os.Getenv("LOG_LEVEL") // Например: "debug", "info", "warn".

log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithLevelString(level),
)
defer log.Close()
```

Если строка пустая, будет `INFO`. Если строка неизвестная, `New` вернет ошибку,
а `MustNew` сделает panic.

## 6. Поля по умолчанию

Поля, добавленные через `WithField` или `WithFields`, будут в каждой записи
логгера.

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithFormat(logger.FormatJSON),
	logger.WithFields(logger.Fields{
		"service":     "billing-api",
		"environment": "prod",
	}),
)
defer log.Close()

log.Info("invoice created", logger.Fields{
	"invoice_id": 9001,
})
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"invoice created","environment":"prod","service":"billing-api","invoice_id":9001}
```

## 7. Дочерние логгеры

Дочерний логгер наследует настройки родителя и добавляет свои поля. Это удобно
для worker'ов, tenant'ов, request id, trace id.

```go
base := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithFormat(logger.FormatJSON),
	logger.WithField("service", "billing-api"),
)
defer base.Close()

workerLog := base.WithFields(logger.Fields{
	"component": "email-worker",
	"worker_id": "worker-1",
})

workerLog.Info("job started", logger.Fields{"job_id": "job-777"})
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"job started","service":"billing-api","component":"email-worker","worker_id":"worker-1","job_id":"job-777"}
```

## 8. Ошибки и app code

`WithError` и `WithAppCode` добавляют поля к дочернему логгеру. Метод `Error`
делает то же самое для конкретной записи.

```go
err := errors.New("sql: no rows")

log.WithError(err).
	WithAppCode(40401).
	Warn("user was not found", 40401, logger.Fields{
		"user_id": 42,
	})

log.Error("failed to load user", err, 50001, logger.Fields{
	"user_id": 42,
})
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"WARN","message":"user was not found","app_code":40401,"error":"sql: no rows","user_id":42}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"failed to load user","app_code":50001,"error":"sql: no rows","user_id":42}
```

## 9. Группы полей

`WithGroup` группирует следующие поля в объект. В JSON это удобно для больших
структур вроде `request`, `db`, `user`.

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

## 10. Низкоуровневый метод Log

`Log` полезен, если уровень вычисляется динамически.

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

## 11. Context-aware логирование

Логгер можно положить в `context.Context`, а потом доставать в глубине кода.

```go
ctx := log.WithField("request_id", "req-123").IntoContext(context.Background())

func loadUser(ctx context.Context, userID int) {
	// Если в context нет логгера, FromContext вернет глобальный logger.Get().
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

## 12. Проверка Enabled

`Enabled` помогает не собирать дорогие поля, если уровень отключен.

```go
if log.Enabled(context.Background(), logger.LevelDebug) {
	log.Debug("expensive debug data", logger.Fields{
		"payload": buildLargeDebugPayload(),
	})
}
```

Если `LevelDebug` выключен, `buildLargeDebugPayload()` не вызовется.

## 13. Глобальный логгер

Глобальный API удобен для маленьких сервисов и legacy-кода.

```go
func main() {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		panic(err)
	}

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
}
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"global logger started","service":"billing-api"}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"WARN","message":"global warning","service":"billing-api","app_code":2001,"retry":true}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"global error","service":"billing-api","app_code":5001,"error":"boom"}
```

Нюанс API: у метода `log.Debug` сигнатура `Debug(msg, fields)`, а у глобальной
функции `logger.Debug` сигнатура `Debug(msg, appCode, fields)`.

## 14. Init: короткий старый вариант

`Init` оставлен для простого старта. Он принимает уровень, формат времени и путь
к файлу.

```go
err := logger.Init(
	logger.LevelInfo,
	time.RFC3339,
	"logs/app.log",
)
if err != nil {
	panic(err)
}
defer logger.Close()

logger.Info("started through Init", nil)
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00+03:00","level":"INFO","message":"started through Init"}
```

## 15. Настройка времени

`WithTimeFormat` меняет формат поля `timestamp`.

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

## 16. Добавить source

`WithAddSource(true)` добавляет место вызова. Это полезно при отладке, но в
продакшене может быть шумно.

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

## 17. Несколько outputs

`WithOutputs` пишет одну и ту же строку сразу в несколько мест.

```go
file, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
if err != nil {
	panic(err)
}
defer file.Close()

log := logger.MustNew(
	// Лог попадет и в файл, и в stderr.
	logger.WithOutputs(file, os.Stderr),
)
defer log.Close()

log.Info("written to two outputs", nil)
```

В `logs/app.log`:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"written to two outputs"}
```

## 18. Замена или удаление атрибутов

`WithReplaceAttr` позволяет переименовать, изменить или скрыть поле.

```go
log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithReplaceAttr(func(groups []string, attr slog.Attr) slog.Attr {
		// Пример: скрываем поле user_email.
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

## 19. Свой slog.Handler

`WithHandler` полностью заменяет стандартный JSON/text handler. Это нужно, если
у вас уже есть свой handler, например для отправки в очередь или внешнюю систему.

```go
file, err := os.OpenFile("logs/app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
if err != nil {
	panic(err)
}
defer file.Close()

level := &slog.LevelVar{}
level.Set(slog.LevelInfo)

handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
	Level: level,
})

log := logger.MustNew(
	// Когда задан WithHandler, Format/Output/File уже не используются.
	logger.WithHandler(handler),
)

log.Info("custom handler log", nil)
```

В файле будет стандартный формат `slog` без переименования `time/msg`:

```json
{"time":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","msg":"custom handler log"}
```

## 20. Fatal без завершения процесса

По умолчанию `Fatal` вызывает `os.Exit(1)`. В тестах или CLI это можно заменить.

```go
exitCode := 0

log := logger.MustNew(
	logger.WithFile("logs/app.log"),
	logger.WithExitFunc(func(code int) {
		// В реальном приложении так делать обычно не нужно.
		exitCode = code
	}),
)
defer log.Close()

log.Fatal("cannot start service", errors.New("port is busy"), 9001, nil)
_ = exitCode
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"FATAL","message":"cannot start service","app_code":9001,"error":"port is busy"}
```

## 21. Gin: логирование запросов в файл

```go
func main() {
	if err := os.MkdirAll("logs", 0o755); err != nil {
		panic(err)
	}

	log := logger.MustNew(
		logger.WithFile("logs/app.log"),
		logger.WithFormat(logger.FormatJSON),
		logger.WithLevel(logger.LevelInfo),
		logger.WithField("service", "orders-api"),
	)
	defer log.Close()

	router := gin.New()

	// Этот middleware первым создает request logger и кладет его в gin.Context.
	router.Use(middleware.StructuredLogHandler(
		middleware.WithRequestLogger(log),
		middleware.WithHeaderLogging("X-Request-ID"),
		middleware.WithRequestBodyLogging(16*1024),
	))

	// Этот middleware должен идти после StructuredLogHandler.
	router.Use(middleware.ErrorHandler(
		middleware.WithErrorLogger(log),
	))

	router.POST("/orders", func(c *gin.Context) {
		// Берем логгер, уже обогащенный method/path/ip/request_id.
		middleware.RequestLogger(c).Info("inside handler", logger.Fields{
			"phase": "validation",
		})

		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	_ = router.Run(":8080")
}
```

При запросе:

```http
POST /orders?token=secret&search=bag
X-Request-ID: req-123
Content-Type: application/json

{"sku":"bag"}
```

В файле будет примерно:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"inside handler","service":"orders-api","ip":"127.0.0.1","method":"POST","path":"/orders","protocol":"HTTP/1.1","request_id":"req-123","user_agent":"","phase":"validation"}
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"request completed","service":"orders-api","ip":"127.0.0.1","method":"POST","path":"/orders","protocol":"HTTP/1.1","request_id":"req-123","user_agent":"","status":201,"latency_ms":1,"response_bytes":11,"route":"/orders","query":"search=bag&token=%5BREDACTED%5D","headers":{"X-Request-Id":"req-123"},"body":"{\"sku\":\"bag\"}"}
```

## 22. Gin middleware: основные опции

```go
router.Use(middleware.StructuredLogHandler(
	middleware.WithRequestLogger(log), // Использовать ваш file logger.

	middleware.WithRequestMessages(
		"http request completed", // Сообщение для успешных запросов.
		"http request failed",    // Сообщение для ошибок и 4xx/5xx.
	),

	middleware.WithSkipPaths("/healthz", "/metrics"), // Не логировать шумные пути.

	middleware.WithHeaderLogging("X-Request-ID", "Authorization"),
	// Authorization попадет в лог как [REDACTED] по умолчанию.

	middleware.WithCookieLogging("session"),
	// session cookie тоже будет [REDACTED] по умолчанию.

	middleware.WithRequestBodyLogging(4096),
	// Тело пишется только до лимита. multipart/form-data не логируется.

	middleware.WithRedactedQueryParams("email"),
	// Дополнительно скрыть query-параметр email.

	middleware.WithRequestExtraFields(func(c *gin.Context) logger.Fields {
		return logger.Fields{
			"tenant_id": c.GetHeader("X-Tenant-ID"),
		}
	}),
))
```

Пример строки:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"http request completed","status":200,"latency_ms":3,"path":"/orders","query":"email=%5BREDACTED%5D","headers":{"Authorization":"[REDACTED]","X-Request-Id":"req-123"},"cookies":{"session":"[REDACTED]"},"tenant_id":"acme"}
```

## 23. Свой уровень для HTTP-запросов

```go
router.Use(middleware.StructuredLogHandler(
	middleware.WithRequestLogger(log),
	middleware.WithRequestLevelResolver(func(c *gin.Context) logger.Level {
		// Например, 404 считаем INFO, чтобы не засорять WARN.
		if c.Writer.Status() == http.StatusNotFound {
			return logger.LevelInfo
		}
		if c.Writer.Status() >= 500 {
			return logger.LevelError
		}
		if c.Writer.Status() >= 400 {
			return logger.LevelWarn
		}
		return logger.LevelInfo
	}),
))
```

## 24. Форматирование body

`WithBodyFormatter` полезен, если нельзя писать тело целиком.

```go
router.Use(middleware.StructuredLogHandler(
	middleware.WithRequestLogger(log),
	middleware.WithRequestBodyLogging(16*1024),
	middleware.WithBodyFormatter(func(c *gin.Context, body []byte) any {
		// В лог пишем не тело, а только размер.
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

## 25. AppError: безопасные ошибки для клиента и полезные логи

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

## 26. Конструкторы AppError

```go
apperror.BadRequestError(err, 1001, "Bad input")       // HTTP 400.
apperror.UnauthorizedError(err, 1002, "Unauthorized")  // HTTP 401.
apperror.ForbiddenError(err, 1003, "Forbidden")        // HTTP 403.
apperror.NotFoundError(err, 1004, "Not found")         // HTTP 404.
apperror.ConflictError(err, 1005, "Conflict")          // HTTP 409.
apperror.SystemError(err, 9000, "Internal error")      // HTTP 500.
apperror.CustomError(err, 418, 41801, "Custom error")  // Любой статус.
```

Если message передать пустым, большинство конструкторов подставит стандартное
сообщение.

## 27. Достать AppError из wrapped error

```go
err := apperror.NotFoundError(errors.New("sql: no rows"), 40401, "User not found")
wrapped := errors.Join(errors.New("load user failed"), err)

if appErr, ok := apperror.As(wrapped); ok {
	log.Error(appErr.Message, appErr.Err, appErr.AppCode, appErr.LogFields())
}
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"User not found","app_code":40401,"error":"sql: no rows"}
```

## 28. Необработанная ошибка в Gin

```go
router.GET("/boom", func(c *gin.Context) {
	c.Error(errors.New("database is down"))
})
```

Клиент получит:

```json
{"app_code":9999,"message":"Internal Server Error"}
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"Unhandled system error","app_code":9999,"error":"database is down","status":500}
```

## 29. Настройка fallback-ошибки

```go
router.Use(middleware.ErrorHandler(
	middleware.WithErrorLogger(log),
	middleware.WithUnhandledError(9500, "Unexpected server error"),
))
```

Клиент получит:

```json
{"app_code":9500,"message":"Unexpected server error"}
```

## 30. Логирование body в ErrorHandler

```go
router.Use(middleware.ErrorHandler(
	middleware.WithErrorLogger(log),
	middleware.WithErrorBodyLogging(4096),
))
```

Если handler прочитал body и добавил ошибку, в файле будет:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"Invalid user ID","app_code":1001,"error":"invalid id","body":"{\"id\":\"bad\"}","status":400}
```

## 31. WebSocket error helper

```go
func onWebSocketError(conn *websocket.Conn, err error) {
	middleware.HandleWebSocketError(conn, err, "websocket operation failed")
}
```

Если ошибка не `AppError`, она будет превращена в системную ошибку с кодом
`9999`. Клиенту по WebSocket отправится:

```json
{"type":"error","message":"WebSocket internal error","app_code":9999}
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"ERROR","message":"websocket operation failed","app_code":9999,"error":"some websocket error","protocol":"websocket"}
```

## 32. Slog напрямую

Если нужен доступ к стандартному `*slog.Logger`, используйте `Slog()`.

```go
slogLogger := log.Slog()
slogLogger.Info("native slog call", "source", "third-party")
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"native slog call","source":"third-party"}
```

## 33. MergeFields

`MergeFields` удобно использовать, когда общие поля нужно соединить с локальными.
Если ключ повторяется, победит последнее значение.

```go
baseFields := logger.Fields{"service": "billing-api", "env": "prod"}
requestFields := logger.Fields{"request_id": "req-123", "env": "stage"}

log.Info("merged fields", logger.MergeFields(baseFields, requestFields))
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"merged fields","env":"stage","request_id":"req-123","service":"billing-api"}
```

## 34. Рекомендуемый шаблон для сервиса

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

func main() {
	log, err := NewAppLogger()
	if err != nil {
		panic(err)
	}
	defer log.Close()

	logger.SetDefault(log)

	log.Info("application started", logger.Fields{
		"pid": os.Getpid(),
	})
}
```

В файле:

```json
{"timestamp":"2026-05-11T13:00:00.000000000+03:00","level":"INFO","message":"application started","environment":"prod","service":"orders-api","pid":12345}
```

## 35. Частые ошибки

- Не забывайте `os.MkdirAll("logs", 0o755)`: библиотека создает файл, но не
  создает директорию.
- Не забывайте `Close()`, если используете `WithFile`, иначе файл может быть
  закрыт только при завершении процесса.
- Не пишите чувствительные данные в `Fields`: middleware маскирует популярные
  secrets, но ваши кастомные поля нужно фильтровать самостоятельно.
- Не включайте body logging без лимита и без понимания, что в body может быть
  пароль, токен или персональные данные.
- Для Gin ставьте `StructuredLogHandler` перед `ErrorHandler`.
- Для продакшена лучше `FormatJSON`, для локальной ручной отладки можно
  `FormatText`.
- Если нужен строго один файл без stdout, используйте `os.OpenFile` +
  `WithOutput(file)`, а не `WithFile`.
