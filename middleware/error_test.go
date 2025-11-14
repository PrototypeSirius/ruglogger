package middleware_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PrototypeSirius/ruglogger/apperror"
	"github.com/PrototypeSirius/ruglogger/logger"
	"github.com/PrototypeSirius/ruglogger/middleware"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRouter(logOutput *bytes.Buffer) *gin.Engine {
	logger.Init(logger.WithOutput(logOutput))
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.ErrorHandler())
	return router
}

func TestErrorHandler_HandlesAppError(t *testing.T) {
	logger.ResetForTest()
	logBuffer := new(bytes.Buffer)
	router := setupTestRouter(logBuffer)
	router.GET("/test-app-error", func(c *gin.Context) {
		testErr := errors.New("underlying db error")
		appErr := apperror.BadRequestError(testErr, 1001, "Неверный ID пользователя")
		_ = c.Error(appErr)
	})
	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-app-error", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusBadRequest, recorder.Code, "HTTP статус должен быть 400 Bad Request")
	expectedJSON := `{"message":"Неверный ID пользователя", "app_code":1001}`
	assert.JSONEq(t, expectedJSON, recorder.Body.String(), "Тело ответа JSON не соответствует ожидаемому")
	logOutput := logBuffer.String()
	require.NotEmpty(t, logOutput, "Лог не должен быть пустым")
	assert.Contains(t, logOutput, `"level":"error"`, "Уровень лога должен быть 'error'")
	assert.Contains(t, logOutput, `"msg":"Неверный ID пользователя"`, "Сообщение лога неверное")
	assert.Contains(t, logOutput, `"error":"underlying db error"`, "Системная ошибка должна быть в логе")
	assert.Contains(t, logOutput, `"app_code":1001`, "Код приложения должен быть в логе")
}

func TestErrorHandler_HandlesUnexpectedError(t *testing.T) {
	logger.ResetForTest()
	logBuffer := new(bytes.Buffer)
	router := setupTestRouter(logBuffer)

	router.GET("/test-unexpected-error", func(c *gin.Context) {
		_ = c.Error(errors.New("что-то пошло не так"))
	})

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/test-unexpected-error", nil)

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code, "HTTP статус должен быть 500 Internal Server Error")
	expectedJSON := `{"message":"Internal Server Error", "app_code":9404}`
	assert.JSONEq(t, expectedJSON, recorder.Body.String(), "Тело ответа JSON не соответствует ожидаемому")
	logOutput := logBuffer.String()
	require.NotEmpty(t, logOutput, "Лог не должен быть пустым")
	assert.Contains(t, logOutput, `"level":"error"`, "Уровень лога должен быть 'error'")
	assert.Contains(t, logOutput, `"msg":"Unhandled internal error"`, "Сообщение лога неверное")
	assert.Contains(t, logOutput, `"error":"что-то пошло не так"`, "Оригинальная ошибка должна быть в логе")
}

func setupStructuredLogRouter(logOutput *bytes.Buffer) *gin.Engine {
	logger.Init(logger.WithOutput(logOutput))
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.StructuredLogHandler())
	return router
}

func TestAPIStructuredLog_LogsAllFields(t *testing.T) {
	logger.ResetForTest()
	logBuffer := new(bytes.Buffer)
	router := setupStructuredLogRouter(logBuffer)

	router.GET("/log-test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	recorder := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/log-test?param1=value1&param2=value2", nil)
	req.AddCookie(&http.Cookie{Name: "session_id", Value: "abc-123"})
	req.AddCookie(&http.Cookie{Name: "theme", Value: "dark"})
	req.Header.Set("User-Agent", "Go-Test")

	router.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)

	logOutput := logBuffer.String()
	require.NotEmpty(t, logOutput, "Лог не должен быть пустым")

	var logData map[string]interface{}
	err := json.Unmarshal([]byte(logOutput), &logData)
	require.NoError(t, err, "Лог должен быть в валидном JSON формате")

	assert.Equal(t, "info", logData["level"])
	assert.Equal(t, "Request processed", logData["msg"])
	assert.Equal(t, float64(200), logData["status_code"])
	assert.Equal(t, "/log-test", logData["path"])
	assert.Equal(t, "Go-Test", logData["user_agent"])
	assert.Equal(t, "param1=value1&param2=value2", logData["query"])

	cookies, ok := logData["cookies"].(map[string]interface{})
	require.True(t, ok, "Поле 'cookies' должно быть объектом")
	assert.Equal(t, "abc-123", cookies["session_id"])
	assert.Equal(t, "dark", cookies["theme"])

	expectedCookies := map[string]interface{}{"session_id": "abc-123", "theme": "dark"}
	assert.Equal(t, expectedCookies, logData["cookies"])
}

func TestAPIStructuredLog_LogsRequestBody(t *testing.T) {
	logger.ResetForTest()
	logBuffer := new(bytes.Buffer)
	router := setupStructuredLogRouter(logBuffer)
	router.POST("/log-body", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	recorder := httptest.NewRecorder()

	requestBody := `{"name":"Sirius"}`
	req, _ := http.NewRequest(http.MethodPost, "/log-body", strings.NewReader(requestBody))

	router.ServeHTTP(recorder, req)

	logOutput := logBuffer.String()
	require.NotEmpty(t, logOutput)

	var logData map[string]interface{}
	err := json.Unmarshal([]byte(logOutput), &logData)
	require.NoError(t, err)

	assert.Equal(t, requestBody, logData["request_body"])
}
