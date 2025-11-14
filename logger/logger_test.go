package logger

import (
	"bytes"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInit_SingletonPattern проверяет, что `Init` действительно выполняется только один раз.
func TestInit_SingletonPattern(t *testing.T) {
	buffer1 := new(bytes.Buffer)
	buffer2 := new(bytes.Buffer)
	log = nil
	once = sync.Once{}
	Init(WithOutput(buffer1), WithLevel("debug"))
	Get().Info("первый вызов")
	Init(WithOutput(buffer2), WithLevel("error"))
	Get().Info("второй вызов")
	assert.Contains(t, buffer1.String(), "первый вызов")
	assert.Contains(t, buffer1.String(), "второй вызов")
	assert.Empty(t, buffer2.String())
}

func TestLogger_JSONFormat(t *testing.T) {
	buffer := new(bytes.Buffer)
	log = nil
	once = sync.Once{}
	Init(WithOutput(buffer), WithLevel("info"))
	log := Get()
	log.WithField("user_id", 123).Info("тестовое сообщение")
	var result map[string]interface{}
	err := json.Unmarshal(buffer.Bytes(), &result)
	require.NoError(t, err, "Лог должен быть в валидном JSON формате")
	assert.Equal(t, "info", result["level"])
	assert.Equal(t, "тестовое сообщение", result["msg"])
	assert.Equal(t, float64(123), result["user_id"])
}
