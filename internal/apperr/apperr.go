package apperr

import (
	"fmt"
	"net/http"
)

type Kind int

const (
	KindInternalServerError Kind = iota // * Set to 0 so that zero values trigger Internal Server Error
	KindBadRequest
	KindUnauthorized
	KindForbidden
	KindNotFound
	KindConflict
	KindTooManyRequests
	KindServiceUnavailable
)

type Apperr struct {
	Kind    Kind
	Message string
	Err     error
	Data    any
}

// * To satisfy the error interface
func (a *Apperr) Error() string {
	if a.Err != nil {
		return a.Message + ": " + a.Err.Error()
	}
	return a.Message
}

// * To access the underlying error
func (a *Apperr) Unwrap() error { return a.Err }

// * Return correct status codes based on Kind
func (k Kind) HTTPStatus() int {
	switch k {
	case KindBadRequest:
		return http.StatusBadRequest
	case KindUnauthorized:
		return http.StatusUnauthorized
	case KindForbidden:
		return http.StatusForbidden
	case KindNotFound:
		return http.StatusNotFound
	case KindConflict:
		return http.StatusConflict
	case KindTooManyRequests:
		return http.StatusTooManyRequests
	case KindServiceUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// * Helpers
func InternalServerError(msg string, err error) error {
	return &Apperr{Kind: KindInternalServerError, Message: "internal server error", Err: fmt.Errorf("%s: %w", msg, err)}
}
func BadRequest(msg string, data any) error {
	return &Apperr{Kind: KindBadRequest, Message: fmt.Sprintf("bad request: %s", msg), Data: data}
}
func Unauthorized() error {
	return &Apperr{Kind: KindUnauthorized, Message: "unauthorized"}
}
func Forbidden() error {
	return &Apperr{Kind: KindForbidden, Message: "forbidden"}
}
func NotFound(msg string) error {
	return &Apperr{Kind: KindNotFound, Message: fmt.Sprintf("bad request: %s", msg)}
}
func Conflict(data any) error {
	return &Apperr{Kind: KindConflict, Message: "conflict", Data: data}
}
func TooManyRequests() error {
	return &Apperr{Kind: KindTooManyRequests, Message: "too many requests"}
}
func ServiceUnavailable() error {
	return &Apperr{Kind: KindServiceUnavailable, Message: "service unavailable"}
}
