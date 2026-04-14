package middleware

import (
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	logger "github.com/PrototypeSirius/ruglogger/ruglog"
	"github.com/gin-gonic/gin"
)

const RequestLoggerKey = "ruglogger.request_logger"

var (
	defaultRedactedHeaders = normalizeKeySet(
		"authorization",
		"cookie",
		"set-cookie",
		"x-api-key",
		"x-auth-token",
	)
	defaultRedactedCookies = normalizeKeySet(
		"session",
		"sessionid",
		"session_id",
		"csrf",
		"csrf_token",
		"refresh_token",
	)
	defaultRedactedQueryParams = normalizeKeySet(
		"password",
		"token",
		"access_token",
		"refresh_token",
		"secret",
		"api_key",
	)
)

func RequestLogger(c *gin.Context) *logger.Logger {
	if c == nil {
		return logger.Get()
	}
	if value, ok := c.Get(RequestLoggerKey); ok {
		if requestLogger, ok := value.(*logger.Logger); ok && requestLogger != nil {
			return requestLogger
		}
	}
	return logger.FromContext(c.Request.Context())
}

func attachRequestLogger(c *gin.Context, requestLogger *logger.Logger) {
	c.Set(RequestLoggerKey, requestLogger)
	c.Request = c.Request.WithContext(requestLogger.IntoContext(c.Request.Context()))
}

func baseRequestFields(c *gin.Context, requestIDHeaders []string) logger.Fields {
	fields := logger.Fields{
		"protocol":   c.Request.Proto,
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"ip":         c.ClientIP(),
		"user_agent": c.Request.UserAgent(),
	}
	if route := c.FullPath(); route != "" {
		fields["route"] = route
	}
	if requestID := firstNonEmptyHeader(c.Request.Header, requestIDHeaders); requestID != "" {
		fields["request_id"] = requestID
	}
	return fields
}

func firstNonEmptyHeader(header http.Header, names []string) string {
	for _, name := range names {
		value := strings.TrimSpace(header.Get(name))
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeKeySet(values ...string) map[string]struct{} {
	if len(values) == 0 {
		return map[string]struct{}{}
	}
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		key := strings.ToLower(strings.TrimSpace(value))
		if key == "" {
			continue
		}
		set[key] = struct{}{}
	}
	return set
}

func mergeKeySets(sets ...map[string]struct{}) map[string]struct{} {
	merged := map[string]struct{}{}
	for _, set := range sets {
		for key := range set {
			merged[key] = struct{}{}
		}
	}
	return merged
}

func collectHeaders(header http.Header, allowlist map[string]struct{}, redacted map[string]struct{}) logger.Fields {
	if len(header) == 0 {
		return nil
	}

	result := logger.Fields{}
	keys := make([]string, 0, len(header))
	for key := range header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		normalized := strings.ToLower(key)
		if len(allowlist) > 0 {
			if _, ok := allowlist[normalized]; !ok {
				continue
			}
		}
		values := header.Values(key)
		if len(values) == 0 {
			continue
		}
		if _, ok := redacted[normalized]; ok {
			result[key] = "[REDACTED]"
			continue
		}
		if len(values) == 1 {
			result[key] = values[0]
			continue
		}
		copied := make([]string, len(values))
		copy(copied, values)
		result[key] = copied
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func collectCookies(cookies []*http.Cookie, allowlist map[string]struct{}, redacted map[string]struct{}) logger.Fields {
	if len(cookies) == 0 {
		return nil
	}

	result := logger.Fields{}
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		normalized := strings.ToLower(cookie.Name)
		if len(allowlist) > 0 {
			if _, ok := allowlist[normalized]; !ok {
				continue
			}
		}
		if _, ok := redacted[normalized]; ok {
			result[cookie.Name] = "[REDACTED]"
			continue
		}
		result[cookie.Name] = cookie.Value
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

func sanitizeQuery(rawQuery string, redacted map[string]struct{}) string {
	if strings.TrimSpace(rawQuery) == "" {
		return ""
	}

	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return rawQuery
	}

	for key := range values {
		if _, ok := redacted[strings.ToLower(key)]; ok {
			values[key] = []string{"[REDACTED]"}
		}
	}

	return values.Encode()
}

type bodyCaptureReadCloser struct {
	reader    io.ReadCloser
	limit     int64
	mu        sync.Mutex
	buffer    []byte
	truncated bool
}

func newBodyCaptureReadCloser(reader io.ReadCloser, limit int64) *bodyCaptureReadCloser {
	return &bodyCaptureReadCloser{
		reader: reader,
		limit:  limit,
		buffer: make([]byte, 0, limit),
	}
}

func (b *bodyCaptureReadCloser) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if n > 0 && b.limit > 0 {
		b.mu.Lock()
		defer b.mu.Unlock()

		remaining := b.limit - int64(len(b.buffer))
		if remaining > 0 {
			chunk := p[:n]
			if int64(len(chunk)) > remaining {
				b.buffer = append(b.buffer, chunk[:remaining]...)
				b.truncated = true
			} else {
				b.buffer = append(b.buffer, chunk...)
			}
		} else {
			b.truncated = true
		}
	}
	return n, err
}

func (b *bodyCaptureReadCloser) Close() error {
	return b.reader.Close()
}

func (b *bodyCaptureReadCloser) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()

	copied := make([]byte, len(b.buffer))
	copy(copied, b.buffer)
	return copied
}

func (b *bodyCaptureReadCloser) Truncated() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.truncated
}

func shouldCaptureRequestBody(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	if contentType == "" {
		return true
	}
	return !strings.HasPrefix(contentType, "multipart/form-data")
}
