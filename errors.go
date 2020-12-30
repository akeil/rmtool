package rm

import (
	"fmt"
	"net/http"
)

// Wrap wraps an error by prepending additional text.
// The text can contain formatting parameters.
func Wrap(err error, msg string, v ...interface{}) error {
	msg = fmt.Sprintf(msg, v...)
	return fmt.Errorf("%v: %v", msg, err)
}

type notFound struct {
	message string
}

// NewNotFound creates a new "not found" error.
func NewNotFound(s string, v ...interface{}) error {
	return asNotFound(fmt.Errorf(s, v...))
}

func (n notFound) Error() string {
	return n.message
}

func asNotFound(e error) error {
	return notFound{fmt.Sprintf("Not found: %v", e)}
}

// IsNotFound checks if the given error is a "not found" error.
func IsNotFound(err error) bool {
	_, ok := err.(notFound)
	return ok
}

type validationError struct {
	message string
}

func (v validationError) Error() string {
	return v.message
}

// NewValidationError creates an error of from the given format string.
func NewValidationError(msg string, v ...interface{}) error {
	return validationError{fmt.Sprintf(msg, v...)}
}

// ExpectOK checks if the given http response has status "200 - OK"
// and returns an error with the given message if not.
func ExpectOK(res *http.Response, msg string) error {
	return ExpectStatus(res, http.StatusOK, msg)
}

// ExpectStatus checks if the given http response has the expected status
// and returns an error with the given message if not.
func ExpectStatus(res *http.Response, expected int, msg string) error {
	code := res.StatusCode

	if code == expected {
		return nil
	}

	if msg != "" {
		msg = msg + ": "
	}

	// specific types for selected error codes
	switch code {
	case http.StatusNotFound:
		return NewNotFound("%vgot HTTP status %v", msg, code)
	}

	// unspecified errors
	return fmt.Errorf("%vgot HTTP status code %v", msg, code)
}
