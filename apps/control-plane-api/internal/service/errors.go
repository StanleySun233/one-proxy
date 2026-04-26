package service

import "net/http"

type Error struct {
	Status  int
	Code    string
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func newError(status int, code string) *Error {
	return &Error{
		Status:  status,
		Code:    code,
		Message: code,
	}
}

func invalidInput(code string) *Error {
	return newError(http.StatusBadRequest, code)
}

func unauthorized(code string) *Error {
	return newError(http.StatusUnauthorized, code)
}

func internalFailure(code string) *Error {
	return newError(http.StatusInternalServerError, code)
}
