# ruglogger

Лёгкая библиотека структурированного логирования для Go с небольшим API, логгерами уровня запроса для Gin и ошибками приложения, которые безопасны для клиента, но полезны в логах.

## Что изменилось

- Настраиваемые экземпляры логгера, а не только один глобальный экземпляр
- Вывод в JSON или текстовом формате через `log/slog`
- Дочерние логгеры с наследованием полей
- Изменение уровня логирования во время работы
- Хелперы для логирования с поддержкой `context`
- Middleware для Gin с маскированием чувствительных данных, перехватом тела запроса, идентификатором запроса и внедрением логгера на уровень запроса
- Ошибки приложения с HTTP-статусом, кодом приложения и необязательными публичными `details`
- Тесты для логгера, middleware и сценариев обработки ошибок

## Пакеты

- `ruglog`: структурированный логгер
- `rugerror`: ошибки приложения
- `middleware`: промежуточный слой для Gin для логирования запросов и обработки ошибок

## Быстрый старт

```go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperror "github.com/PrototypeSirius/ruglogger/rugerror"
	logger "github.com/PrototypeSirius/ruglogger/ruglog"
	"github.com/PrototypeSirius/ruglogger/middleware"
)

func main() {
	log := logger.MustNew(
		logger.WithFormat(logger.FormatJSON),
		logger.WithLevel(logger.LevelDebug),
		logger.WithField("service", "billing-api"),
	)

	logger.SetDefault(log)

	router := gin.New()
	router.Use(middleware.StructuredLogHandler(
		middleware.WithRequestLogger(log),
		middleware.WithHeaderLogging("X-Request-ID"),
		middleware.WithRequestBodyLogging(16*1024),
	))
	router.Use(middleware.ErrorHandler(
		middleware.WithErrorLogger(log),
	))

	router.POST("/orders", func(c *gin.Context) {
		middleware.RequestLogger(c).Info("создание заказа", logger.Fields{
			"phase": "validation",
		})

		c.Error(
			apperror.BadRequestError(nil, 1001, "Некорректные данные заказа").WithDetails(map[string]any{
				"field": "items",
			}),
		)
	})

	_ = http.ListenAndServe(":8080", router)
}
```

## Глобальный логгер

Используйте классический глобальный API, если нужен максимально простой старт:

```go
if err := logger.InitWithOptions(
	logger.WithFormat(logger.FormatJSON),
	logger.WithLevel(logger.LevelInfo),
	logger.WithFile("app.log"),
); err != nil {
	panic(err)
}

logger.Info("приложение запущено", logger.Fields{
	"version": "1.2.3",
})
```

## Локальные экземпляры логгера

Создавайте независимые логгеры, когда нужна изоляция в тестах, воркерах или мультиарендных сервисах:

```go
log := logger.MustNew(
	logger.WithFormat(logger.FormatText),
	logger.WithField("worker", "emails"),
)

tenantLog := log.WithFields(logger.Fields{
	"tenant_id": "acme",
})

tenantLog.Info("задача запущена", nil)
```

## Поведение middleware по умолчанию

`StructuredLogHandler()` намеренно сделан безопаснее и гибче старой версии:

- Параметры строки запроса логируются по умолчанию, при этом распространённые секреты маскируются
- Заголовки, cookie и тело запроса включаются явно через опции
- Перехват тела запроса не “съедает” его до того, как обработчик успеет его прочитать
- Логгер, привязанный к запросу, прокидывается и в `gin.Context`, и в контекст запроса

Полезные опции:

- `WithRequestLogger(log)`
- `WithRequestBodyLogging(limit)`
- `WithHeaderLogging(names...)`
- `WithCookieLogging(names...)`
- `WithRedactedHeaders(names...)`
- `WithRedactedCookies(names...)`
- `WithRedactedQueryParams(names...)`
- `WithSkipPaths(paths...)`
- `WithRequestExtraFields(func(*gin.Context) logger.Fields)`

Внутри обработчиков:

```go
middleware.RequestLogger(c).Info("внутри обработчика", logger.Fields{
	"user_id": 42,
})
```

## Ошибки приложения

`rugerror.AppError` разделяет:

- внутреннюю причину для логов
- безопасное сообщение для клиента
- HTTP-статус
- стабильный код приложения
- необязательные публичные `details`

Примеры:

```go
err := apperror.NotFoundError(sql.ErrNoRows, 2004, "Пользователь не найден")

validationErr := apperror.BadRequestError(nil, 1001, "Некорректные входные данные").WithDetails(map[string]any{
	"field": "email",
})
```

## Тестирование

Сейчас в библиотеке уже есть тесты для:

- конфигурации логгера и дочерних логгеров
- `fatal`-логирования без завершения тестового процесса
- логирования запросов Gin и маскировки чувствительных данных
- ответов с `AppError` и резервной обработки `500`

Запуск:

```bash
go test ./...
```
