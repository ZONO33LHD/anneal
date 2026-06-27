// Package errors defines application errors with a stable code, so layers above
// can branch on the kind of failure without depending on concrete error types.
package errors

import (
	"errors"
	"fmt"
)

// Code is a coarse, stable error classification.
type Code string

const (
	CodeBadRequest Code = "BAD_REQUEST"
	CodeNotFound   Code = "NOT_FOUND"
	CodeConflict   Code = "CONFLICT"
	CodeInternal   Code = "INTERNAL"
)

// AppError carries a code, a human message, and an optional wrapped cause.
type AppError struct {
	Code    Code
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New builds an AppError.
func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap builds an AppError around a cause.
func Wrap(err error, code Code, message string) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// CodeOf extracts the code, defaulting to internal.
func CodeOf(err error) Code {
	if ae, ok := errors.AsType[*AppError](err); ok {
		return ae.Code
	}
	return CodeInternal
}
