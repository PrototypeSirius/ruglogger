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

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		lastErr := c.Errors.Last().Err
		var appErr *apperror.AppError

		if errors.As(lastErr, &appErr) {
			logger.LogOnError(appErr, appErr.Message, logrus.Fields{
				"path":     c.Request.URL.Path,
				"method":   c.Request.Method,
				"app_code": appErr.AppCode,
			})
			c.AbortWithStatusJSON(appErr.HTTPStatus, appErr)
		} else {
			logger.LogOnError(lastErr, "Unhandled internal error", logrus.Fields{
				"path":     c.Request.URL.Path,
				"method":   c.Request.Method,
				"app_code": 9404,
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message":  "Internal Server Error",
				"app_code": 9404,
			})
		}
	}
}

type WebSocketErrorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	AppCode int    `json:"app_code"`
}

func HandleWebSocketError(conn *websocket.Conn, err error, fields logrus.Fields) {
	if err == nil {
		return
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		appErr = apperror.SystemError(err, 9404, "")
	}

	logFields := fields
	if logFields == nil {
		logFields = logrus.Fields{}
	}
	logFields["protocol"] = "websocket"
	logger.LogOnError(appErr, appErr.Message, logFields)

	response := WebSocketErrorResponse{
		Type:    "error",
		Message: appErr.Message,
		AppCode: appErr.AppCode,
	}

	if writeErr := conn.WriteJSON(response); writeErr != nil {
		logger.LogOnError(writeErr, "Failed to send error message over WebSocket")
	}
}
