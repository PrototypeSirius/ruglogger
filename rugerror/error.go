package apperror

import (
	"encoding/json"
	"net/http"
)

type AppError struct {
	Err        error  `json:"-"`
	Message    string `json:"message"`
	HTTPStatus int    `json:"-"`
	AppCode    int    `json:"app_code"`
}

func New(err error, httpStatus int, appCode int, message string) *AppError {
	return &AppError{
		Err:        err,
		Message:    message,
		HTTPStatus: httpStatus,
		AppCode:    appCode,
	}
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Message string `json:"message"`
		AppCode int    `json:"app_code"`
	}{
		Message: e.Message,
		AppCode: e.AppCode,
	})
}

func SystemError(err error, appCode int, message string) *AppError {
	if message == "" {
		message = "Internal Server Error"
	}
	return New(err, http.StatusInternalServerError, appCode, message)
}
