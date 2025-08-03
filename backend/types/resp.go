package types

import "strings"

type Resp[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
}

func RespWithOk[T any](data T, msg ...string) Resp[T] {
	message := "ok"
	if len(msg) > 0 {
		message = strings.Join(msg, ",")
	}
	return Resp[T]{
		Code:    0,
		Message: message,
		Data:    data,
	}
}

func RespWithErrorAndData[T any](code int, data T, msg ...string) Resp[T] {
	message := "error"
	if len(msg) > 0 {
		message = strings.Join(msg, ",")
	}
	return Resp[T]{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func RespWithError(code int, msg ...string) Resp[any] {
	message := "error"
	if len(msg) > 0 {
		message = strings.Join(msg, ",")
	}
	return Resp[any]{
		Code:    code,
		Message: message,
		Data:    nil,
	}
}
