package apperror

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAppError_ErrorInterface проверяет, что AppError корректно реализует интерфейс error.
func TestAppError_ErrorInterface(t *testing.T) {

	originalErr := errors.New("оригинальная ошибка")
	appErr := New(originalErr, http.StatusBadRequest, 101, "публичное сообщение")
	assert.Equal(t, "оригинальная ошибка", appErr.Error())
	assert.True(t, errors.Is(appErr, originalErr))
}

func TestAppError_MarshalJSON(t *testing.T) {
	appErr := New(errors.New("секретная ошибка базы данных"), http.StatusInternalServerError, 100, "Внутренняя ошибка сервера")
	jsonData, err := json.Marshal(appErr)
	require.NoError(t, err)
	expectedJSON := `{"message":"Внутренняя ошибка сервера","app_code":100}`
	assert.JSONEq(t, expectedJSON, string(jsonData))
}
