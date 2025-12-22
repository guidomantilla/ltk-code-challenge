package core

import (
	"encoding/json"
	"errors"
	"fmt"
)

var ErrEventNotFound = errors.New("event not found")

type Error struct {
	Message string   `json:"message,omitempty"`
	Err     []string `json:"err,omitempty"`
}

func NewError(message string, errs ...error) *Error {
	return &Error{
		Message: message,
		Err: func() []string {
			var msgs []string

			for _, err := range errs {
				if err != nil {
					msgs = append(msgs, err.Error())
				}
			}

			return msgs
		}(),
	}
}

func (e *Error) Error() string {
	//nolint:errchkjson
	data, _ := json.Marshal(e)
	return string(data)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	if len(e.Err) == 0 {
		return nil
	}

	errs := make([]error, len(e.Err))
	for i, err := range e.Err {
		errs[i] = fmt.Errorf("%s", err)
	}

	return errors.Join(errs...)
}

func (e *Error) Messages() []string {
	return e.Err
}
