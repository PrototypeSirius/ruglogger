package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/PrototypeSirius/ruglogger/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// maxBodySize определяет максимальный размер тела запроса (в байтах), который будет прочитан для логирования.
// Это мера безопасности, предотвращающая исчерпание памяти сервера при получении очень больших запросов (DoS-атака).
const maxBodySize = 16384 // 16 KB
// StructuredLog - это middleware для Gin, которое логирует каждый запрос и ответ в структурированном JSON-формате.
// В лог включается информация о времени ответа (latency), HTTP-статусе, IP клиента, методе, пути и т.д.
func APIStructuredLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		var requestBody []byte
		if c.Request.Body != nil {
			limitedReader := &io.LimitedReader{R: c.Request.Body, N: maxBodySize}
			requestBody, _ = io.ReadAll(limitedReader)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		log := logger.Get()

		logEntry := log.WithFields(logrus.Fields{
			"status_code":  statusCode,
			"latency_ms":   latency.Milliseconds(),
			"client_ip":    c.ClientIP(),
			"method":       c.Request.Method,
			"path":         path,
			"user_agent":   c.Request.UserAgent(),
			"request_body": string(requestBody),
		})

		if rawQuery != "" {
			logEntry = logEntry.WithField("query", rawQuery)
		}

		if len(c.Errors) > 0 {
			logEntry.Error(c.Errors.String())
		} else {
			logEntry.Info("запрос обработан")
		}
	}
}
