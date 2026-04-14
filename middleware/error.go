package middleware

import (
	"net/http"

	apperror "github.com/PrototypeSirius/ruglogger/rugerror"
	logger "github.com/PrototypeSirius/ruglogger/ruglog"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const defaultInternalAppCode = 9999

type ErrorHandlerOption func(*errorHandlerConfig)

type errorHandlerConfig struct {
	logger          *logger.Logger
	internalMessage string
	internalAppCode int
	includeBody     bool
	bodyLimit       int64
	extraFields     func(*gin.Context) logger.Fields
	requestIDs      []string
}

func defaultErrorHandlerConfig() errorHandlerConfig {
	return errorHandlerConfig{
		logger:          logger.Get(),
		internalMessage: "Internal Server Error",
		internalAppCode: defaultInternalAppCode,
		bodyLimit:       defaultBodyLimit,
		requestIDs:      []string{"X-Request-ID", "X-Correlation-ID"},
	}
}

func WithErrorLogger(log *logger.Logger) ErrorHandlerOption {
	return func(cfg *errorHandlerConfig) {
		if log != nil {
			cfg.logger = log
		}
	}
}

func WithUnhandledError(code int, message string) ErrorHandlerOption {
	return func(cfg *errorHandlerConfig) {
		if code != 0 {
			cfg.internalAppCode = code
		}
		if message != "" {
			cfg.internalMessage = message
		}
	}
}

func WithErrorBodyLogging(limit int64) ErrorHandlerOption {
	return func(cfg *errorHandlerConfig) {
		cfg.includeBody = true
		if limit > 0 {
			cfg.bodyLimit = limit
		}
	}
}

func WithErrorExtraFields(extractor func(*gin.Context) logger.Fields) ErrorHandlerOption {
	return func(cfg *errorHandlerConfig) {
		cfg.extraFields = extractor
	}
}

func ErrorHandler(opts ...ErrorHandlerOption) gin.HandlerFunc {
	cfg := defaultErrorHandlerConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return func(c *gin.Context) {
		var bodyCapture *bodyCaptureReadCloser
		if cfg.includeBody && cfg.bodyLimit > 0 && c.Request.Body != nil && shouldCaptureRequestBody(c.GetHeader("Content-Type")) {
			bodyCapture = newBodyCaptureReadCloser(c.Request.Body, cfg.bodyLimit)
			c.Request.Body = bodyCapture
		}

		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		requestLogger := cfg.logger
		if value, ok := c.Get(RequestLoggerKey); ok {
			if scopedLogger, ok := value.(*logger.Logger); ok && scopedLogger != nil {
				requestLogger = scopedLogger
			}
		}
		if requestLogger == nil {
			requestLogger = logger.Get()
		}

		fields := logger.Fields{
			"status": c.Writer.Status(),
		}
		if route := c.FullPath(); route != "" {
			fields["route"] = route
		}
		if _, ok := c.Get(RequestLoggerKey); !ok {
			requestLogger = requestLogger.WithFields(baseRequestFields(c, cfg.requestIDs))
		}
		if bodyCapture != nil {
			body := bodyCapture.Bytes()
			if len(body) > 0 {
				fields["body"] = string(body)
			}
			if bodyCapture.Truncated() {
				fields["body_truncated"] = true
			}
		}
		if cfg.extraFields != nil {
			fields = logger.MergeFields(fields, cfg.extraFields(c))
		}

		lastErr := c.Errors.Last().Err
		if appErr, ok := apperror.As(lastErr); ok {
			fields["status"] = appErr.HTTPStatus
			fields = logger.MergeFields(fields, appErr.LogFields())
			requestLogger.Error(appErr.Message, appErr.Err, appErr.AppCode, fields)
			if !c.Writer.Written() {
				c.AbortWithStatusJSON(appErr.HTTPStatus, appErr)
				return
			}
			c.Abort()
			return
		}

		fields["status"] = http.StatusInternalServerError
		requestLogger.Error("Unhandled system error", lastErr, cfg.internalAppCode, fields)
		if !c.Writer.Written() {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message":  cfg.internalMessage,
				"app_code": cfg.internalAppCode,
			})
			return
		}
		c.Abort()
	}
}

type WebSocketErrorResponse struct {
	Type    string         `json:"type,omitempty"`
	Message string         `json:"message"`
	AppCode int            `json:"app_code"`
	Details map[string]any `json:"details,omitempty"`
}

func HandleWebSocketError(conn *websocket.Conn, err error, contextMsg string) {
	if err == nil {
		return
	}

	appErr, ok := apperror.As(err)
	if !ok {
		appErr = apperror.SystemError(err, defaultInternalAppCode, "WebSocket internal error")
	}

	message := contextMsg
	if message == "" {
		message = appErr.Message
	}

	fields := logger.Fields{
		"protocol": "websocket",
	}
	fields = logger.MergeFields(fields, appErr.LogFields())

	logger.Get().Error(message, appErr.Err, appErr.AppCode, fields)
	if conn == nil {
		return
	}

	response := WebSocketErrorResponse{
		Type:    "error",
		Message: appErr.Message,
		AppCode: appErr.AppCode,
		Details: appErr.Details,
	}
	if writeErr := conn.WriteJSON(response); writeErr != nil {
		logger.Get().Error(
			"Failed to send error message over WebSocket",
			writeErr,
			appErr.AppCode,
			logger.Fields{
				"protocol":        "websocket",
				"websocket_error": appErr.Error(),
			},
		)
	}
}
