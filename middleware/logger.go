package middleware

import (
	"strings"
	"time"

	logger "github.com/PrototypeSirius/ruglogger/ruglog"
	"github.com/gin-gonic/gin"
)

const defaultBodyLimit int64 = 16 * 1024

type StructuredLogOption func(*structuredLogConfig)

type structuredLogConfig struct {
	logger             *logger.Logger
	successMessage     string
	errorMessage       string
	bodyLimit          int64
	includeQuery       bool
	includeHeaders     bool
	includeCookies     bool
	includeBody        bool
	skip               func(*gin.Context) bool
	extraFields        func(*gin.Context) logger.Fields
	levelResolver      func(*gin.Context) logger.Level
	bodyFormatter      func(*gin.Context, []byte) any
	requestIDHeaders   []string
	headerAllowlist    map[string]struct{}
	cookieAllowlist    map[string]struct{}
	redactedHeaders    map[string]struct{}
	redactedCookies    map[string]struct{}
	redactedQueryParms map[string]struct{}
}

func defaultStructuredLogConfig() structuredLogConfig {
	return structuredLogConfig{
		logger:             logger.Get(),
		successMessage:     "request completed",
		errorMessage:       "request failed",
		bodyLimit:          defaultBodyLimit,
		includeQuery:       true,
		requestIDHeaders:   []string{"X-Request-ID", "X-Correlation-ID"},
		redactedHeaders:    mergeKeySets(defaultRedactedHeaders),
		redactedCookies:    mergeKeySets(defaultRedactedCookies),
		redactedQueryParms: mergeKeySets(defaultRedactedQueryParams),
		levelResolver: func(c *gin.Context) logger.Level {
			if len(c.Errors) > 0 || c.Writer.Status() >= 500 {
				return logger.LevelError
			}
			if c.Writer.Status() >= 400 {
				return logger.LevelWarn
			}
			return logger.LevelInfo
		},
	}
}

func WithRequestLogger(log *logger.Logger) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		if log != nil {
			cfg.logger = log
		}
	}
}

func WithRequestMessages(successMessage string, errorMessage string) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		if successMessage != "" {
			cfg.successMessage = successMessage
		}
		if errorMessage != "" {
			cfg.errorMessage = errorMessage
		}
	}
}

func WithRequestSkipper(skip func(*gin.Context) bool) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		cfg.skip = skip
	}
}

func WithSkipPaths(paths ...string) StructuredLogOption {
	pathSet := map[string]struct{}{}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		pathSet[path] = struct{}{}
	}
	return func(cfg *structuredLogConfig) {
		previous := cfg.skip
		cfg.skip = func(c *gin.Context) bool {
			if previous != nil && previous(c) {
				return true
			}
			_, ok := pathSet[c.Request.URL.Path]
			return ok
		}
	}
}

func WithQueryLogging() StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		cfg.includeQuery = true
	}
}

func WithoutQueryLogging() StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		cfg.includeQuery = false
	}
}

func WithHeaderLogging(names ...string) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		cfg.includeHeaders = true
		if len(names) == 0 {
			cfg.headerAllowlist = nil
			return
		}
		cfg.headerAllowlist = normalizeKeySet(names...)
	}
}

func WithCookieLogging(names ...string) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		cfg.includeCookies = true
		if len(names) == 0 {
			cfg.cookieAllowlist = nil
			return
		}
		cfg.cookieAllowlist = normalizeKeySet(names...)
	}
}

func WithRequestBodyLogging(limit int64) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		cfg.includeBody = true
		if limit > 0 {
			cfg.bodyLimit = limit
		}
	}
}

func WithBodyFormatter(formatter func(*gin.Context, []byte) any) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		cfg.bodyFormatter = formatter
	}
}

func WithRequestExtraFields(extractor func(*gin.Context) logger.Fields) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		cfg.extraFields = extractor
	}
}

func WithRequestLevelResolver(resolver func(*gin.Context) logger.Level) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		if resolver != nil {
			cfg.levelResolver = resolver
		}
	}
}

func WithRequestIDHeaders(names ...string) StructuredLogOption {
	return func(cfg *structuredLogConfig) {
		if len(names) == 0 {
			cfg.requestIDHeaders = nil
			return
		}
		copied := make([]string, len(names))
		copy(copied, names)
		cfg.requestIDHeaders = copied
	}
}

func WithRedactedHeaders(names ...string) StructuredLogOption {
	set := normalizeKeySet(names...)
	return func(cfg *structuredLogConfig) {
		cfg.redactedHeaders = mergeKeySets(cfg.redactedHeaders, set)
	}
}

func WithRedactedCookies(names ...string) StructuredLogOption {
	set := normalizeKeySet(names...)
	return func(cfg *structuredLogConfig) {
		cfg.redactedCookies = mergeKeySets(cfg.redactedCookies, set)
	}
}

func WithRedactedQueryParams(names ...string) StructuredLogOption {
	set := normalizeKeySet(names...)
	return func(cfg *structuredLogConfig) {
		cfg.redactedQueryParms = mergeKeySets(cfg.redactedQueryParms, set)
	}
}

func StructuredLogHandler(opts ...StructuredLogOption) gin.HandlerFunc {
	cfg := defaultStructuredLogConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return func(c *gin.Context) {
		if cfg.skip != nil && cfg.skip(c) {
			c.Next()
			return
		}

		start := time.Now()
		requestLogger := cfg.logger.WithFields(baseRequestFields(c, cfg.requestIDHeaders))
		attachRequestLogger(c, requestLogger)

		var bodyCapture *bodyCaptureReadCloser
		if cfg.includeBody && cfg.bodyLimit > 0 && c.Request.Body != nil && shouldCaptureRequestBody(c.GetHeader("Content-Type")) {
			bodyCapture = newBodyCaptureReadCloser(c.Request.Body, cfg.bodyLimit)
			c.Request.Body = bodyCapture
		}

		c.Next()

		fields := logger.Fields{
			"status":     c.Writer.Status(),
			"latency_ms": time.Since(start).Milliseconds(),
		}
		if size := c.Writer.Size(); size >= 0 {
			fields["response_bytes"] = size
		}

		if route := c.FullPath(); route != "" {
			fields["route"] = route
		}

		if cfg.includeQuery {
			if query := sanitizeQuery(c.Request.URL.RawQuery, cfg.redactedQueryParms); query != "" {
				fields["query"] = query
			}
		}

		if cfg.includeHeaders {
			if headers := collectHeaders(c.Request.Header, cfg.headerAllowlist, cfg.redactedHeaders); len(headers) > 0 {
				fields["headers"] = headers
			}
		}

		if cfg.includeCookies {
			if cookies := collectCookies(c.Request.Cookies(), cfg.cookieAllowlist, cfg.redactedCookies); len(cookies) > 0 {
				fields["cookies"] = cookies
			}
		}

		if bodyCapture != nil {
			body := bodyCapture.Bytes()
			if len(body) > 0 {
				if cfg.bodyFormatter != nil {
					if formatted := cfg.bodyFormatter(c, body); formatted != nil {
						fields["body"] = formatted
					}
				} else {
					fields["body"] = string(body)
				}
			}
			if bodyCapture.Truncated() {
				fields["body_truncated"] = true
			}
		}

		if cfg.extraFields != nil {
			fields = logger.MergeFields(fields, cfg.extraFields(c))
		}

		var lastErr error
		if len(c.Errors) > 0 {
			lastErr = c.Errors.Last().Err
			fields["error_count"] = len(c.Errors)
			errorsText := make([]string, 0, len(c.Errors))
			for _, entry := range c.Errors {
				if entry == nil || entry.Err == nil {
					continue
				}
				errorsText = append(errorsText, entry.Err.Error())
			}
			if len(errorsText) > 0 {
				fields["errors"] = errorsText
			}
		}

		level := cfg.levelResolver(c)
		message := cfg.successMessage
		if level >= logger.LevelWarn || lastErr != nil {
			message = cfg.errorMessage
		}

		requestLogger.Log(level, message, lastErr, 0, fields)
	}
}
