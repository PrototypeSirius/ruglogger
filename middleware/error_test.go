package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PrototypeSirius/ruglogger/apperror"
	"github.com/PrototypeSirius/ruglogger/logger"
	"github.com/stretchr/testify/assert"

	"github.com/gin-gonic/gin"
)

func TestErrorHandler(t *testing.T) {
	logger.Init(logger.WithOutput(io.Discard))
	gin.SetMode(gin.TestMode)

	// Создаем тестовый сценарий для ожидаемой ошибки (AppError)
	t.Run("should handle AppError and return specific code and message", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		_, r := gin.CreateTestContext(recorder)
		r.Use(APIErrorHandler())
		r.GET("/test", func(ctx *gin.Context) {
			_ = ctx.Error(apperror.BadRequestError("неверный ID", errors.New("validation error")))
		})
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		r.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assert.JSONEq(t, `{"message":"неверный ID", "code":400}`, recorder.Body.String())
	})
	t.Run("should handle unexpected error and return 500", func(t *testing.T) {
		recorder := httptest.NewRecorder()
		_, r := gin.CreateTestContext(recorder)
		r.Use(APIErrorHandler())
		r.GET("/test-panic", func(ctx *gin.Context) {
			_ = ctx.Error(errors.New("что-то пошло не так"))
		})
		req, _ := http.NewRequest(http.MethodGet, "/test-panic", nil)
		r.ServeHTTP(recorder, req)
		assert.Equal(t, http.StatusInternalServerError, recorder.Code)
		assert.JSONEq(t, `{"message":"Внутренняя ошибка сервера", "code":500}`, recorder.Body.String())
	})
}
