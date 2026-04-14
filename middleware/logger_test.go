package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	logger "github.com/PrototypeSirius/ruglogger/ruglog"
	"github.com/gin-gonic/gin"
)

func TestStructuredLogHandlerAddsRequestLoggerAndRedactsSensitiveData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	log := logger.MustNew(
		logger.WithOutput(&output),
		logger.WithFormat(logger.FormatJSON),
	)

	router := gin.New()
	router.Use(StructuredLogHandler(
		WithRequestLogger(log),
		WithHeaderLogging("Authorization", "X-Request-ID"),
		WithCookieLogging(),
		WithRequestBodyLogging(64),
	))
	router.POST("/orders", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		RequestLogger(c).Info("inside handler", logger.Fields{
			"phase": "handler",
		})
		c.Status(http.StatusCreated)
	})

	req := httptest.NewRequest(http.MethodPost, "/orders?token=secret&search=bag", strings.NewReader(`{"sku":"bag"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer super-secret")
	req.Header.Set("X-Request-ID", "req-123")
	req.AddCookie(&http.Cookie{Name: "session", Value: "cookie-secret"})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	records := decodeLogLines(t, output.String())
	if len(records) != 2 {
		t.Fatalf("expected 2 log lines, got %d: %s", len(records), output.String())
	}

	inside := records[0]
	if inside["message"] != "inside handler" {
		t.Fatalf("unexpected inner message: %#v", inside["message"])
	}
	if inside["request_id"] != "req-123" {
		t.Fatalf("request logger did not inherit request id: %#v", inside["request_id"])
	}
	if inside["path"] != "/orders" {
		t.Fatalf("request logger did not inherit path: %#v", inside["path"])
	}

	requestLog := records[1]
	if requestLog["message"] != "request completed" {
		t.Fatalf("unexpected request message: %#v", requestLog["message"])
	}
	if requestLog["status"] != float64(http.StatusCreated) {
		t.Fatalf("unexpected request status: %#v", requestLog["status"])
	}
	if requestLog["body"] != `{"sku":"bag"}` {
		t.Fatalf("unexpected request body: %#v", requestLog["body"])
	}
	if requestLog["query"] != "search=bag&token=%5BREDACTED%5D" {
		t.Fatalf("unexpected query log: %#v", requestLog["query"])
	}

	headers, ok := requestLog["headers"].(map[string]any)
	if !ok {
		t.Fatalf("expected headers map, got %#v", requestLog["headers"])
	}
	if headers["Authorization"] != "[REDACTED]" {
		t.Fatalf("authorization should be redacted: %#v", headers["Authorization"])
	}
	if headers["X-Request-Id"] != "req-123" && headers["X-Request-ID"] != "req-123" {
		t.Fatalf("request id header missing: %#v", headers)
	}

	cookies, ok := requestLog["cookies"].(map[string]any)
	if !ok {
		t.Fatalf("expected cookies map, got %#v", requestLog["cookies"])
	}
	if cookies["session"] != "[REDACTED]" {
		t.Fatalf("session cookie should be redacted: %#v", cookies["session"])
	}
}

func decodeLogLines(t *testing.T, raw string) []map[string]any {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("failed to decode log line %q: %v", line, err)
		}
		records = append(records, record)
	}
	return records
}
