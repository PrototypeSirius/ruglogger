package apperror

import (
	"encoding/json"
	"errors"
	"net/http"
)

type AppError struct {
	Err        error          `json:"-"`
	Message    string         `json:"message"`
	HTTPStatus int            `json:"-"`
	AppCode    int            `json:"app_code"`
	Details    map[string]any `json:"details,omitempty"`
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

func (e *AppError) Status() int {
	return e.HTTPStatus
}

func (e *AppError) WithDetails(details map[string]any) *AppError {
	if len(details) == 0 {
		return e
	}
	if e.Details == nil {
		e.Details = map[string]any{}
	}
	for key, value := range details {
		e.Details[key] = value
	}
	return e
}

func (e *AppError) PublicFields() map[string]any {
	fields := map[string]any{
		"message":  e.Message,
		"app_code": e.AppCode,
	}
	if len(e.Details) > 0 {
		fields["details"] = e.Details
	}
	return fields
}

func (e *AppError) LogFields() map[string]any {
	fields := map[string]any{
		"app_code": e.AppCode,
	}
	if len(e.Details) > 0 {
		fields["details"] = e.Details
	}
	return fields
}

func (e *AppError) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.PublicFields())
}

func As(err error) (*AppError, bool) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		return nil, false
	}
	return appErr, true
}

func SystemError(err error, appCode int, message string) *AppError {
	if message == "" {
		message = "Internal Server Error"
	}
	return New(err, http.StatusInternalServerError, appCode, message)
}

func BadRequestError(err error, appCode int, message string) *AppError {
	if message == "" {
		message = "Bad Request"
	}
	return New(err, http.StatusBadRequest, appCode, message)
}

func UnauthorizedError(err error, appCode int, message string) *AppError {
	if message == "" {
		message = "Unauthorized"
	}
	return New(err, http.StatusUnauthorized, appCode, message)
}

func ForbiddenError(err error, appCode int, message string) *AppError {
	if message == "" {
		message = "Forbidden"
	}
	return New(err, http.StatusForbidden, appCode, message)
}

func NotFoundError(err error, appCode int, message string) *AppError {
	if message == "" {
		message = "Resource not found"
	}
	return New(err, http.StatusNotFound, appCode, message)
}

func ConflictError(err error, appCode int, message string) *AppError {
	if message == "" {
		message = "Conflict"
	}
	return New(err, http.StatusConflict, appCode, message)
}

func CustomError(err error, httpStatus int, appCode int, message string) *AppError {
	return New(err, httpStatus, appCode, message)
}
