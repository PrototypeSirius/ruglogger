package apperror

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/PrototypeSirius/ruglogger/logger"
	"github.com/sirupsen/logrus"
)

// AppError представляет кастомный тип ошибки.
// Он инкапсулирует оригинальную системную ошибку, публичное сообщение и HTTP-код.
type AppError struct {
	// Err - это исходная, системная ошибка (например, от базы данных или другой службы).
	// Это поле не должно показываться конечному пользователю, но оно обязательно для логирования.
	// Тег `json:"-"` предотвращает его попадание в JSON-ответ клиенту.
	Err error `json:"-"`
	// Message - это безопасное, публичное сообщение для клиента.
	Message string `json:"message"`
	// Code - это HTTP-статус, который будет отправлен в ответе.
	Code int `json:"code"`
}

// New - это конструктор для создания нового экземпляра AppError.
func New(err error, code int, message string) *AppError {
	return &AppError{
		Err:     err,
		Message: message,
		Code:    code,
	}
}

// Error реализует стандартный интерфейс `error`, что позволяет использовать AppError везде,
// где ожидается обычная ошибка. Возвращает сообщение системной ошибки для логирования.
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// Unwrap предоставляет совместимость со стандартными функциями `errors.Is` и `errors.As`.
// Это позволяет "развернуть" нашу ошибку и докопаться до исходной системной ошибки `e.Err`.
func (e *AppError) Unwrap() error {
	return e.Err
}

// MarshalJSON реализует интерфейс `json.Marshaler`.
// Этот метод гарантирует, что при сериализации AppError в JSON будет сформирован
// только "чистый", безопасный для клиента ответ, без внутренней информации об ошибке.
func (e *AppError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	}{
		Message: e.Message,
		Code:    e.Code,
	})
}

// SystemError - это хелпер для создания стандартной внутренней ошибки сервера (500).
// Скрывает от клиента детали системной ошибки, возвращая общее сообщение.
func SystemError(message string, err error) *AppError {
	if message == "" {
		message = "Internal Server Error"
	}
	return New(err, http.StatusInternalServerError, message)
}
func BadRequestError(message string, err error) *AppError {
	if message == "" {
		message = "Incorrect request"
	}
	return New(err, http.StatusBadRequest, message)
}
func UnauthorizedError(message string, err error) *AppError {
	if message == "" {
		message = "Unauthorized"
	}
	return New(err, http.StatusUnauthorized, message)
}

func ErrorChecker(appErr *AppError) {
	if appErr.Err != nil {
		log.Printf("[ERROR] %s: %v", appErr.Message, appErr.Err)
	}
}

func FatalErrorChecker(appErr *AppError) {
	if appErr.Err != nil {
		log.Fatalf("[ERROR] %s: %v", appErr.Message, appErr.Err)
	}
}

func LogErrorHandler(err error, fields logrus.Fields) *AppError {

	if err == nil {
		return nil
	}

	log := logger.Get()
	var appErr *AppError
	logEntry := log.WithFields(fields)
	if errors.As(err, &appErr) {
		logEntry.WithFields(logrus.Fields{
			"error_code":   appErr.Code,
			"system_error": appErr.Err,
		}).Error(appErr.Message)

		return appErr
	}

	logEntry.WithField("error", err).Error("Non-standard error")
	return nil
}
