// Package errors は安定したコードを持つアプリケーションエラーを定義する。これにより
// 上位レイヤは具象エラー型に依存せず、失敗の種類で分岐できる。
package errors

import (
	"errors"
	"fmt"
)

// Code は粗粒度で安定したエラー分類である。
type Code string

const (
	CodeBadRequest Code = "BAD_REQUEST"
	CodeNotFound   Code = "NOT_FOUND"
	CodeConflict   Code = "CONFLICT"
	CodeInternal   Code = "INTERNAL"
)

// AppError はコード、人間向けメッセージ、および任意のラップされた原因を保持する。
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

// New は AppError を生成する。
func New(code Code, message string) *AppError {
	return &AppError{Code: code, Message: message}
}

// Wrap は原因をラップした AppError を生成する。
func Wrap(err error, code Code, message string) *AppError {
	return &AppError{Code: code, Message: message, Err: err}
}

// CodeOf はコードを取り出す。該当しない場合は internal にフォールバックする。
func CodeOf(err error) Code {
	if ae, ok := errors.AsType[*AppError](err); ok {
		return ae.Code
	}
	return CodeInternal
}
