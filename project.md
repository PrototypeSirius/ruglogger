### DIRECTORY ./ FOLDER STRUCTURE ###
DIR apperror/
    FILE error.go
    FILE error_test.go
FILE go.mod
FILE go.sum
DIR logger/
    FILE logger.go
    FILE logger_test.go
DIR middleware/
    FILE error.go
    FILE error_test.go
    FILE logger.go
FILE project.md
FILE README.md
### DIRECTORY ./ FOLDER STRUCTURE ###

### DIRECTORY ./ FLATTENED CONTENT ###
### ./apperror\error.go BEGIN ###
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

### ./apperror\error.go END ###

### ./apperror\error_test.go BEGIN ###
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
	expectedJSON := `{"message":"Внутренняя ошибка сервера","code":500}`
	assert.JSONEq(t, expectedJSON, string(jsonData))
}

### ./apperror\error_test.go END ###

### ./flatten-rust.exe BEGIN ###
[Binary file skipped: ./flatten-rust.exe]
### ./flatten-rust.exe END ###

### ./go.mod BEGIN ###
module github.com/PrototypeSirius/ruglogger

go 1.24.2

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/gorilla/websocket v1.5.3
	github.com/sirupsen/logrus v1.9.3
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/bytedance/sonic v1.14.0 // indirect
	github.com/bytedance/sonic/loader v0.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/gin-contrib/sse v1.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.27.0 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/goccy/go-yaml v1.18.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/quic-go/qpack v0.5.1 // indirect
	github.com/quic-go/quic-go v0.54.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.3.0 // indirect
	go.uber.org/mock v0.5.0 // indirect
	golang.org/x/arch v0.20.0 // indirect
	golang.org/x/crypto v0.40.0 // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/net v0.42.0 // indirect
	golang.org/x/sync v0.16.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.27.0 // indirect
	golang.org/x/tools v0.34.0 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

### ./go.mod END ###

### ./go.sum BEGIN ###
github.com/bytedance/sonic v1.14.0 h1:/OfKt8HFw0kh2rj8N0F6C/qPGRESq0BbaNZgcNXXzQQ=
github.com/bytedance/sonic v1.14.0/go.mod h1:WoEbx8WTcFJfzCe0hbmyTGrfjt8PzNEBdxlNUO24NhA=
github.com/bytedance/sonic/loader v0.3.0 h1:dskwH8edlzNMctoruo8FPTJDF3vLtDT0sXZwvZJyqeA=
github.com/bytedance/sonic/loader v0.3.0/go.mod h1:N8A3vUdtUebEY2/VQC0MyhYeKUFosQU6FxH2JmUe6VI=
github.com/cloudwego/base64x v0.1.6 h1:t11wG9AECkCDk5fMSoxmufanudBtJ+/HemLstXDLI2M=
github.com/cloudwego/base64x v0.1.6/go.mod h1:OFcloc187FXDaYHvrNIjxSe8ncn0OOM8gEHfghB2IPU=
github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/gabriel-vasile/mimetype v1.4.8 h1:FfZ3gj38NjllZIeJAmMhr+qKL8Wu+nOoI3GqacKw1NM=
github.com/gabriel-vasile/mimetype v1.4.8/go.mod h1:ByKUIKGjh1ODkGM1asKUbQZOLGrPjydw3hYPU2YU9t8=
github.com/gin-contrib/sse v1.1.0 h1:n0w2GMuUpWDVp7qSpvze6fAu9iRxJY4Hmj6AmBOU05w=
github.com/gin-contrib/sse v1.1.0/go.mod h1:hxRZ5gVpWMT7Z0B0gSNYqqsSCNIJMjzvm6fqCz9vjwM=
github.com/gin-gonic/gin v1.11.0 h1:OW/6PLjyusp2PPXtyxKHU0RbX6I/l28FTdDlae5ueWk=
github.com/gin-gonic/gin v1.11.0/go.mod h1:+iq/FyxlGzII0KHiBGjuNn4UNENUlKbGlNmc+W50Dls=
github.com/go-playground/assert/v2 v2.2.0 h1:JvknZsQTYeFEAhQwI4qEt9cyV5ONwRHC+lYKSsYSR8s=
github.com/go-playground/assert/v2 v2.2.0/go.mod h1:VDjEfimB/XKnb+ZQfWdccd7VUvScMdVu0Titje2rxJ4=
github.com/go-playground/locales v0.14.1 h1:EWaQ/wswjilfKLTECiXz7Rh+3BjFhfDFKv/oXslEjJA=
github.com/go-playground/locales v0.14.1/go.mod h1:hxrqLVvrK65+Rwrd5Fc6F2O76J/NuW9t0sjnWqG1slY=
github.com/go-playground/universal-translator v0.18.1 h1:Bcnm0ZwsGyWbCzImXv+pAJnYK9S473LQFuzCbDbfSFY=
github.com/go-playground/universal-translator v0.18.1/go.mod h1:xekY+UJKNuX9WP91TpwSH2VMlDf28Uj24BCp08ZFTUY=
github.com/go-playground/validator/v10 v10.27.0 h1:w8+XrWVMhGkxOaaowyKH35gFydVHOvC0/uWoy2Fzwn4=
github.com/go-playground/validator/v10 v10.27.0/go.mod h1:I5QpIEbmr8On7W0TktmJAumgzX4CA1XNl4ZmDuVHKKo=
github.com/goccy/go-json v0.10.2 h1:CrxCmQqYDkv1z7lO7Wbh2HN93uovUHgrECaO5ZrCXAU=
github.com/goccy/go-json v0.10.2/go.mod h1:6MelG93GURQebXPDq3khkgXZkazVtN9CRI+MGFi0w8I=
github.com/goccy/go-yaml v1.18.0 h1:8W7wMFS12Pcas7KU+VVkaiCng+kG8QiFeFwzFb+rwuw=
github.com/goccy/go-yaml v1.18.0/go.mod h1:XBurs7gK8ATbW4ZPGKgcbrY1Br56PdM69F7LkFRi1kA=
github.com/google/go-cmp v0.7.0 h1:wk8382ETsv4JYUZwIsn6YpYiWiBsYLSJiTsyBybVuN8=
github.com/google/go-cmp v0.7.0/go.mod h1:pXiqmnSA92OHEEa9HXL2W4E7lf9JzCmGVUdgjX3N/iU=
github.com/google/gofuzz v1.0.0/go.mod h1:dBl0BpW6vV/+mYPU4Po3pmUjxk6FQPldtuIdl/M65Eg=
github.com/gorilla/websocket v1.5.3 h1:saDtZ6Pbx/0u+bgYQ3q96pZgCzfhKXGPqt7kZ72aNNg=
github.com/gorilla/websocket v1.5.3/go.mod h1:YR8l580nyteQvAITg2hZ9XVh4b55+EU/adAjf1fMHhE=
github.com/json-iterator/go v1.1.12 h1:PV8peI4a0ysnczrg+LtxykD8LfKY9ML6u2jnxaEnrnM=
github.com/json-iterator/go v1.1.12/go.mod h1:e30LSqwooZae/UwlEbR2852Gd8hjQvJoHmT4TnhNGBo=
github.com/klauspost/cpuid/v2 v2.3.0 h1:S4CRMLnYUhGeDFDqkGriYKdfoFlDnMtqTiI/sFzhA9Y=
github.com/klauspost/cpuid/v2 v2.3.0/go.mod h1:hqwkgyIinND0mEev00jJYCxPNVRVXFQeu1XKlok6oO0=
github.com/leodido/go-urn v1.4.0 h1:WT9HwE9SGECu3lg4d/dIA+jxlljEa1/ffXKmRjqdmIQ=
github.com/leodido/go-urn v1.4.0/go.mod h1:bvxc+MVxLKB4z00jd1z+Dvzr47oO32F/QSNjSBOlFxI=
github.com/mattn/go-isatty v0.0.20 h1:xfD0iDuEKnDkl03q4limB+vH+GxLEtL/jb4xVJSWWEY=
github.com/mattn/go-isatty v0.0.20/go.mod h1:W+V8PltTTMOvKvAeJH7IuucS94S2C6jfK/D7dTCTo3Y=
github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421 h1:ZqeYNhU3OHLH3mGKHDcjJRFFRrJa6eAM5H+CtDdOsPc=
github.com/modern-go/concurrent v0.0.0-20180228061459-e0a39a4cb421/go.mod h1:6dJC0mAP4ikYIbvyc7fijjWJddQyLn8Ig3JB5CqoB9Q=
github.com/modern-go/reflect2 v1.0.2 h1:xBagoLtFs94CBntxluKeaWgTMpvLxC4ur3nMaC9Gz0M=
github.com/modern-go/reflect2 v1.0.2/go.mod h1:yWuevngMOJpCy52FWWMvUC8ws7m/LJsjYzDa0/r8luk=
github.com/pelletier/go-toml/v2 v2.2.4 h1:mye9XuhQ6gvn5h28+VilKrrPoQVanw5PMw/TB0t5Ec4=
github.com/pelletier/go-toml/v2 v2.2.4/go.mod h1:2gIqNv+qfxSVS7cM2xJQKtLSTLUE9V8t9Stt+h56mCY=
github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
github.com/quic-go/qpack v0.5.1 h1:giqksBPnT/HDtZ6VhtFKgoLOWmlyo9Ei6u9PqzIMbhI=
github.com/quic-go/qpack v0.5.1/go.mod h1:+PC4XFrEskIVkcLzpEkbLqq1uCoxPhQuvK5rH1ZgaEg=
github.com/quic-go/quic-go v0.54.0 h1:6s1YB9QotYI6Ospeiguknbp2Znb/jZYjZLRXn9kMQBg=
github.com/quic-go/quic-go v0.54.0/go.mod h1:e68ZEaCdyviluZmy44P6Iey98v/Wfz6HCjQEm+l8zTY=
github.com/sirupsen/logrus v1.9.3 h1:dueUQJ1C2q9oE3F7wvmSGAaVtTmUizReu6fjN8uqzbQ=
github.com/sirupsen/logrus v1.9.3/go.mod h1:naHLuLoDiP4jHNo9R0sCBMtWGeIprob74mVsIT4qYEQ=
github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
github.com/stretchr/objx v0.4.0/go.mod h1:YvHI0jy2hoMjB+UWwv71VJQ9isScKT/TqJzVSSt89Yw=
github.com/stretchr/objx v0.5.0/go.mod h1:Yh+to48EsGEfYuaHDzXPcE3xhTkx73EhmCGUpEOglKo=
github.com/stretchr/testify v1.3.0/go.mod h1:M5WIy9Dh21IEIfnGCwXGc5bZfKNJtfHm1UVUgZn+9EI=
github.com/stretchr/testify v1.7.0/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.7.1/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
github.com/stretchr/testify v1.8.0/go.mod h1:yNjHg4UonilssWZ8iaSj1OCr/vHnekPRkoO+kdMU+MU=
github.com/stretchr/testify v1.8.1/go.mod h1:w2LPCIKwWwSfY2zedu0+kehJoqGctiVI29o6fzry7u4=
github.com/stretchr/testify v1.11.1 h1:7s2iGBzp5EwR7/aIZr8ao5+dra3wiQyKjjFuvgVKu7U=
github.com/stretchr/testify v1.11.1/go.mod h1:wZwfW3scLgRK+23gO65QZefKpKQRnfz6sD981Nm4B6U=
github.com/twitchyliquid64/golang-asm v0.15.1 h1:SU5vSMR7hnwNxj24w34ZyCi/FmDZTkS4MhqMhdFk5YI=
github.com/twitchyliquid64/golang-asm v0.15.1/go.mod h1:a1lVb/DtPvCB8fslRZhAngC2+aY1QWCk3Cedj/Gdt08=
github.com/ugorji/go/codec v1.3.0 h1:Qd2W2sQawAfG8XSvzwhBeoGq71zXOC/Q1E9y/wUcsUA=
github.com/ugorji/go/codec v1.3.0/go.mod h1:pRBVtBSKl77K30Bv8R2P+cLSGaTtex6fsA2Wjqmfxj4=
go.uber.org/mock v0.5.0 h1:KAMbZvZPyBPWgD14IrIQ38QCyjwpvVVV6K/bHl1IwQU=
go.uber.org/mock v0.5.0/go.mod h1:ge71pBPLYDk7QIi1LupWxdAykm7KIEFchiOqd6z7qMM=
golang.org/x/arch v0.20.0 h1:dx1zTU0MAE98U+TQ8BLl7XsJbgze2WnNKF/8tGp/Q6c=
golang.org/x/arch v0.20.0/go.mod h1:bdwinDaKcfZUGpH09BB7ZmOfhalA8lQdzl62l8gGWsk=
golang.org/x/crypto v0.40.0 h1:r4x+VvoG5Fm+eJcxMaY8CQM7Lb0l1lsmjGBQ6s8BfKM=
golang.org/x/crypto v0.40.0/go.mod h1:Qr1vMER5WyS2dfPHAlsOj01wgLbsyWtFn/aY+5+ZdxY=
golang.org/x/mod v0.25.0 h1:n7a+ZbQKQA/Ysbyb0/6IbB1H/X41mKgbhfv7AfG/44w=
golang.org/x/mod v0.25.0/go.mod h1:IXM97Txy2VM4PJ3gI61r1YEk/gAj6zAHN3AdZt6S9Ww=
golang.org/x/net v0.42.0 h1:jzkYrhi3YQWD6MLBJcsklgQsoAcw89EcZbJw8Z614hs=
golang.org/x/net v0.42.0/go.mod h1:FF1RA5d3u7nAYA4z2TkclSCKh68eSXtiFwcWQpPXdt8=
golang.org/x/sync v0.16.0 h1:ycBJEhp9p4vXvUZNszeOq0kGTPghopOL8q0fq3vstxw=
golang.org/x/sync v0.16.0/go.mod h1:1dzgHSNfp02xaA81J2MS99Qcpr2w7fw1gpm99rleRqA=
golang.org/x/sys v0.0.0-20220715151400-c0bba94af5f8/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.6.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
golang.org/x/sys v0.35.0 h1:vz1N37gP5bs89s7He8XuIYXpyY0+QlsKmzipCbUtyxI=
golang.org/x/sys v0.35.0/go.mod h1:BJP2sWEmIv4KK5OTEluFJCKSidICx8ciO85XgH3Ak8k=
golang.org/x/text v0.27.0 h1:4fGWRpyh641NLlecmyl4LOe6yDdfaYNrGb2zdfo4JV4=
golang.org/x/text v0.27.0/go.mod h1:1D28KMCvyooCX9hBiosv5Tz/+YLxj0j7XhWjpSUF7CU=
golang.org/x/tools v0.34.0 h1:qIpSLOxeCYGg9TrcJokLBG4KFA6d795g0xkBkiESGlo=
golang.org/x/tools v0.34.0/go.mod h1:pAP9OwEaY1CAW3HOmg3hLZC5Z0CCmzjAF2UQMSqNARg=
google.golang.org/protobuf v1.36.9 h1:w2gp2mA27hUeUzj9Ex9FBjsBm40zfaDtEWow293U7Iw=
google.golang.org/protobuf v1.36.9/go.mod h1:fuxRtAxBytpl4zzqUh6/eyUujkJdNiuEkXntxiD/uRU=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405 h1:yhCVgyC4o1eVCa2tZl7eS0r+SDo693bJlVdllGtEeKM=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=

### ./go.sum END ###

### ./logger\logger.go BEGIN ###
package logger

import (
	"io"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	log  *logrus.Logger
	once sync.Once
)

type Option func(*logrus.Logger)

func WithOutput(output io.Writer) Option {
	return func(l *logrus.Logger) {
		l.SetOutput(output)
	}
}

func WithLevel(level string) Option {
	return func(l *logrus.Logger) {
		logLvl, err := logrus.ParseLevel(level)
		if err != nil {
			logLvl = logrus.InfoLevel
		}
		l.SetLevel(logLvl)
	}
}

func Init(opts ...Option) {
	once.Do(func() {
		log = logrus.New()

		log.SetOutput(os.Stdout)
		log.SetLevel(logrus.InfoLevel)
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.999Z07:00",
		})

		for _, opt := range opts {
			opt(log)
		}
	})
}

func Get() *logrus.Logger {
	if log == nil {
		panic("The logger has not been initialized, call logger.Init() in main.go")
	}
	return log
}

func LogOnError(err error, message string, fields ...logrus.Fields) {
	if err == nil {
		return
	}
	entry := Get().WithField("error", err)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Error(message)
}

func FatalOnError(err error, message string, fields ...logrus.Fields) {
	if err == nil {
		return
	}
	entry := Get().WithField("fatal_error", err)
	if len(fields) > 0 {
		entry = entry.WithFields(fields[0])
	}
	entry.Fatal(message) // .Fatal() = .Error() + os.Exit(1)
}

### ./logger\logger.go END ###

### ./logger\logger_test.go BEGIN ###
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

### ./logger\logger_test.go END ###

### ./middleware\error.go BEGIN ###
package middleware

import (
	"errors"
	"net/http"

	"github.com/PrototypeSirius/ruglogger/apperror"
	"github.com/PrototypeSirius/ruglogger/logger"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		lastErr := c.Errors.Last().Err
		var appErr *apperror.AppError

		if errors.As(lastErr, &appErr) {
			logger.LogOnError(appErr, appErr.Message, logrus.Fields{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			c.AbortWithStatusJSON(appErr.HTTPStatus, appErr)
		} else {
			logger.LogOnError(lastErr, "Unhandled internal error", logrus.Fields{
				"path":   c.Request.URL.Path,
				"method": c.Request.Method,
			})
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"message":  "Internal Server Error",
				"app_code": 9404,
			})
		}
	}
}

type WebSocketErrorResponse struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	AppCode int    `json:"app_code"`
}

func HandleWebSocketError(conn *websocket.Conn, err error, fields logrus.Fields) {
	if err == nil {
		return
	}

	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		appErr = apperror.SystemError(err, "")
	}

	logFields := fields
	if logFields == nil {
		logFields = logrus.Fields{}
	}
	logFields["protocol"] = "websocket"
	logger.LogOnError(appErr, appErr.Message, logFields)

	response := WebSocketErrorResponse{
		Type:    "error",
		Message: appErr.Message,
		AppCode: appErr.AppCode,
	}

	if writeErr := conn.WriteJSON(response); writeErr != nil {
		logger.LogOnError(writeErr, "Failed to send error message over WebSocket")
	}
}

### ./middleware\error.go END ###

### ./middleware\error_test.go BEGIN ###
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

	// 2. Проверяем, что было записано в лог.
	logOutput := logBuffer.String()
	require.NotEmpty(t, logOutput, "Лог не должен быть пустым")
	assert.Contains(t, logOutput, `"level":"error"`, "Уровень лога должен быть 'error'")
	assert.Contains(t, logOutput, `"msg":"Неверный ID пользователя"`, "Сообщение лога неверное")
	assert.Contains(t, logOutput, `"error":"underlying db error"`, "Системная ошибка должна быть в логе")
	assert.Contains(t, logOutput, `"app_code":1001`, "Код приложения должен быть в логе")
}

func TestErrorHandler_HandlesUnexpectedError(t *testing.T) {
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
	assert.Contains(t, logOutput, `"msg":"Необработанная внутренняя ошибка"`, "Сообщение лога неверное")
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

### ./middleware\error_test.go END ###

### ./middleware\logger.go BEGIN ###
package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/PrototypeSirius/ruglogger/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const maxBodySize = 16384 // 16 KB

func StructuredLogHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		var requestBody []byte
		if c.Request.Body != nil {
			limitedReader := &io.LimitedReader{R: c.Request.Body, N: maxBodySize}
			requestBody, _ = io.ReadAll(limitedReader)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()
		log := logger.Get()

		logEntry := log.WithFields(logrus.Fields{
			"status_code":  statusCode,
			"latency_ms":   latency.Milliseconds(),
			"client_ip":    c.ClientIP(),
			"method":       c.Request.Method,
			"path":         path,
			"user_agent":   c.Request.UserAgent(),
			"request_body": string(requestBody),
		})

		if rawQuery != "" {
			logEntry = logEntry.WithField("query", rawQuery)
		}

		// if len(c.Request.Cookies()) > 0 {
		// 	ckookie := logrus.Fields{}
		// 	for _, cookie := range c.Request.Cookies() {
		// 		ckookie[cookie.Name] = cookie.Value
		// 	}
		// 	logEntry = logEntry.WithField("cookies", ckookie)
		// }

		if len(c.Errors) > 0 {
			logEntry.Error(c.Errors.String())
		} else {
			logEntry.Info("Request processed")
		}
	}
}

### ./middleware\logger.go END ###

### ./project.md BEGIN ###
### DIRECTORY ./ FOLDER STRUCTURE ###
DIR apperror/
    FILE error.go
    FILE error_test.go
FILE go.mod
FILE go.sum
DIR logger/
    FILE logger.go
    FILE logger_test.go
DIR middleware/
    FILE error.go
    FILE error_test.go
    FILE logger.go
FILE project.md
FILE README.md
### DIRECTORY ./ FOLDER STRUCTURE ###

### DIRECTORY ./ FLATTENED CONTENT ###

### ./project.md END ###

### ./README.md BEGIN ###
# ruglogger
This is a light-weight go library for logging

### ./README.md END ###

### DIRECTORY ./ FLATTENED CONTENT ###
