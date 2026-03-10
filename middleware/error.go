package middleware

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	apperror "github.com/PrototypeSirius/ruglogger/rugerror"
	logger "github.com/PrototypeSirius/ruglogger/ruglog"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var body []byte
		contentType := c.GetHeader("Content-Type")

		if c.Request.Body != nil && !strings.HasPrefix(contentType, "multipart/form-data") {
			body, _ = io.ReadAll(io.LimitReader(c.Request.Body, 16384))
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}
		c.Next()

		if len(c.Errors) == 0 {
			return
		}
		lastErr := c.Errors.Last().Err
		var appErr *apperror.AppError
		fields := map[string]any{
			"protocol":   c.Request.Proto,
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"ip":         c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}
		if len(body) > 0 {
			fields["body"] = string(body)
		}
		if errors.As(lastErr, &appErr) {
			logger.Error(appErr.Message, appErr.Err, appErr.AppCode, fields)
			c.AbortWithStatusJSON(appErr.HTTPStatus, appErr)
		} else {
			logger.Error("Unhandled system error", lastErr, 9999, fields)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message":  "Internal Server Error",
				"app_code": 9999,
			})
		}
	}
}

func HandleWebSocketError(conn *websocket.Conn, err error, contextMsg string) {
	if err == nil {
		return
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		appErr = apperror.SystemError(err, 9999, "WebSocket internal error")
	}
	logger.Error(contextMsg, appErr.Err, appErr.AppCode, map[string]any{
		"protocol": "websocket",
	})
	if errm := conn.WriteJSON(appErr); errm != nil {
		logger.Error("Failed to send error message over WebSocket", errm, appErr.AppCode, map[string]any{
			"protocol":        "websocket",
			"websocket_error": appErr.Err.Error(),
		})
	}

}
