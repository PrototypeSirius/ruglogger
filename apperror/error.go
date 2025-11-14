package apperror

import (
	"encoding/json"
	"net/http"
)

// AppError представляет кастомный тип ошибки.
type AppError struct {
	// Err - исходная, системная ошибка для логирования.
	Err error `json:"-"`
	// Message - публичное, безопасное сообщение для клиента.
	Message string `json:"message"`
	// HTTPStatus - стандартный HTTP-статус (400, 404, 500).
	// Не попадает в JSON-ответ, так как передается в заголовке ответа.
	HTTPStatus int `json:"-"`
	// AppCode - уникальный внутренний код ошибки для удобства отладки и
	// автоматической обработки на клиенте (например, 1001 - "пользователь не найден").
	AppCode int `json:"app_code"`
}

// New - основной конструктор для AppError.
func New(err error, httpStatus int, appCode int, message string) *AppError {
	return &AppError{
		Err:        err,
		Message:    message,
		HTTPStatus: httpStatus,
		AppCode:    appCode,
	}
}

// Error реализует стандартный интерфейс `error`.
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

// Unwrap предоставляет совместимость со стандартными функциями `errors.Is` и `errors.As`.
func (e *AppError) Unwrap() error {
	return e.Err
}

// MarshalJSON настраивает сериализацию AppError в JSON для клиента.
func (e *AppError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message string `json:"message"`
		AppCode int    `json:"app_code"`
	}{
		Message: e.Message,
		AppCode: e.AppCode,
	})
}

// === Хелперы-конструкторы для типовых ошибок ===

// SystemError создает ошибку для внутренних сбоев сервера (HTTP 500).
func SystemError(err error, message string) *AppError {
	if message == "" {
		message = "Internal Server Error"
	}
	return New(err, http.StatusInternalServerError, 9000, message)
}

// BadRequestError создает ошибку для некорректных запросов клиента (HTTP 400).
func BadRequestError(err error, appCode int, message string) *AppError {
	if message == "" {
		message = "Invalid request"
	}
	return New(err, http.StatusBadRequest, appCode, message)
}

// NotFoundError создает ошибку "не найдено" (HTTP 404).
func NotFoundError(err error, appCode int, message string) *AppError {
	if message == "" {
		message = "Resource not found"
	}
	return New(err, http.StatusNotFound, appCode, message)
}

func CustomError(err error, httpStatus int, appCode int, message string) *AppError {
	return New(err, httpStatus, appCode, message)
}
