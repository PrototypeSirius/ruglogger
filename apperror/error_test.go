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
	appErr := New(originalErr, http.StatusBadRequest, "публичное сообщение")
	assert.Equal(t, "оригинальная ошибка", appErr.Error())
	assert.True(t, errors.Is(appErr, originalErr))
}

func TestAppError_MarshalJSON(t *testing.T) {
	appErr := SystemError("ошибка базы данных", errors.New("секретная ошибка базы данных"))
	jsonData, err := json.Marshal(appErr)
	require.NoError(t, err)
	expectedJSON := `{"message":"Внутренняя ошибка сервера","code":500}`
	assert.JSONEq(t, expectedJSON, string(jsonData))
}
