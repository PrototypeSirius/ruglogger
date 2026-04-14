package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperror "github.com/PrototypeSirius/ruglogger/rugerror"
	logger "github.com/PrototypeSirius/ruglogger/ruglog"
	"github.com/gin-gonic/gin"
)

func TestErrorHandlerLogsAppErrorAndReturnsPublicPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	log := logger.MustNew(
		logger.WithOutput(&output),
		logger.WithFormat(logger.FormatJSON),
	)

	router := gin.New()
	router.Use(ErrorHandler(
		WithErrorLogger(log),
		WithErrorBodyLogging(64),
	))
	router.POST("/users", func(c *gin.Context) {
		if _, err := io.ReadAll(c.Request.Body); err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		c.Error(
			apperror.BadRequestError(errors.New("invalid id"), 1001, "Invalid user ID").WithDetails(map[string]any{
				"field": "id",
			}),
		)
	})

	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(`{"id":"bad"}`))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["message"] != "Invalid user ID" {
		t.Fatalf("unexpected response message: %#v", response["message"])
	}
	if response["app_code"] != float64(1001) {
		t.Fatalf("unexpected response app_code: %#v", response["app_code"])
	}

	record := decodeLogLines(t, output.String())[0]
	if record["message"] != "Invalid user ID" {
		t.Fatalf("unexpected log message: %#v", record["message"])
	}
	if record["error"] != "invalid id" {
		t.Fatalf("unexpected log error: %#v", record["error"])
	}
	if record["body"] != `{"id":"bad"}` {
		t.Fatalf("unexpected body in log: %#v", record["body"])
	}
}

func TestErrorHandlerUsesFallbackForUnhandledErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var output bytes.Buffer
	log := logger.MustNew(
		logger.WithOutput(&output),
		logger.WithFormat(logger.FormatJSON),
	)

	router := gin.New()
	router.Use(ErrorHandler(WithErrorLogger(log)))
	router.GET("/boom", func(c *gin.Context) {
		c.Error(errors.New("database is down"))
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response["message"] != "Internal Server Error" {
		t.Fatalf("unexpected response message: %#v", response["message"])
	}
	if response["app_code"] != float64(defaultInternalAppCode) {
		t.Fatalf("unexpected response app_code: %#v", response["app_code"])
	}

	record := decodeLogLines(t, output.String())[0]
	if record["message"] != "Unhandled system error" {
		t.Fatalf("unexpected log message: %#v", record["message"])
	}
	if record["app_code"] != float64(defaultInternalAppCode) {
		t.Fatalf("unexpected app code in log: %#v", record["app_code"])
	}
}
