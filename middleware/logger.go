package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/PrototypeSirius/ruglogger/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const maxBodySize = 16384 // 16 KB

func StructuredLogHandler() gin.HandlerFunc {
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

		if len(c.Request.Header) > 0 {
			hheader := logrus.Fields{}
			for k, v := range c.Request.Header {
				hheader[k] = v
			}
			logEntry = logEntry.WithField("headers", hheader)
		}

		if len(c.Request.Cookies()) > 0 {
			ckookie := logrus.Fields{}
			for _, cookie := range c.Request.Cookies() {
				ckookie[cookie.Name] = cookie.Value
			}
			logEntry = logEntry.WithField("cookies", ckookie)
		}

		if len(c.Errors) > 0 {
			logEntry.Error(c.Errors.String())
		} else {
			logEntry.Info("Request processed")
		}
	}
}
