package domain

import "fmt"

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *Error) Error() string { return e.Message }
func NewError(code, message string, status int) error {
	return &Error{Code: code, Message: message, Status: status}
}
func RevisionError(expected, actual int64) error {
	return NewError("revision_conflict", fmt.Sprintf("expected_revision=%d 与当前 revision=%d 不一致", expected, actual), 409)
}
