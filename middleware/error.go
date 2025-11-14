package middleware

import (
	"errors"
	"net/http"

	"github.com/PrototypeSirius/ruglogger/apperror"
	"github.com/PrototypeSirius/ruglogger/logger"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

// ErrorHandler - это middleware для Gin, которое централизует обработку всех ошибок.
// Оно выполняется после основного обработчика роута, перехватывает ошибки,
// добавленные в контекст Gin через `c.Error()`, логирует их и отправляет
// клиенту стандартизированный JSON-ответ.
func APIErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		lastErr := c.Errors.Last()
		log := logger.Get()

		var appErr *apperror.AppError
		if errors.As(lastErr.Err, &appErr) {
			log.WithFields(map[string]interface{}{
				"error":   appErr.Err,
				"code":    appErr.Code,
				"path":    c.Request.URL.Path,
				"method":  c.Request.Method,
				"user_ip": c.ClientIP(),
			}).Error(appErr.Message) // Публичное сообщение

			c.AbortWithStatusJSON(appErr.Code, appErr)
			return
		}

		log.WithFields(map[string]interface{}{
			"error":   lastErr.Error(),
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
			"user_ip": c.ClientIP(),
		}).Error("Необработанная внутренняя ошибка")

		c.AbortWithStatusJSON(
			http.StatusInternalServerError,
			gin.H{
				"code":    http.StatusInternalServerError,
				"message": "Внутренняя ошибка сервера",
			},
		)
	}
}

type WebSocketErrorResponse struct {
	Type    string `json:"type"`    // Всегда "error" для легкой фильтрации на фронтенде
	Code    int    `json:"code"`    // Код ошибки, аналогичный HTTP-статусу
	Message string `json:"message"` // Публичное сообщение об ошибке
}

// APIWebSocketErrorHandle инкапсулирует полную логику обработки ошибки для WebSocket-соединения.
// Он логирует ошибку, формирует и отправляет стандартизированный JSON-ответ клиенту.
//
// Параметры:
//   - conn: Активное WebSocket-соединение (*websocket.Conn), куда будет отправлен ответ.
//   - err: Ошибка, которую нужно обработать. Если err == nil, функция ничего не делает.
//   - fields: Опциональные поля logrus.Fields для добавления контекста в лог-запись.
func APIWebSocketErrorHandle(conn *websocket.Conn, err error, fields logrus.Fields) {
	if err == nil {
		return
	}
	if fields == nil {
		fields = logrus.Fields{}
	}
	fields["protocol"] = "websocket"
	appErr := apperror.ErrorHandler(err, fields)
	var response WebSocketErrorResponse
	if appErr != nil {
		response = WebSocketErrorResponse{
			Type:    "error",
			Code:    appErr.Code,
			Message: appErr.Message,
		}
	} else {
		response = WebSocketErrorResponse{
			Type:    "error",
			Code:    500,
			Message: "Произошла внутренняя ошибка сервера",
		}
	}
	if writeErr := conn.WriteJSON(response); writeErr != nil {
		logger.Get().WithError(writeErr).Error("Не удалось отправить сообщение об ошибке по WebSocket")
	}
}
