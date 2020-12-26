package rm

import (
	"fmt"
	"net/http"
)

func Wrap(err error, msg string) error {
	return fmt.Errorf("%v: %v", msg, err)
}

type notFound struct {
	message string
}

func NewNotFound(s string, v ...interface{}) error {
	return asNotFound(fmt.Errorf(s, v...))
}

func (n notFound) Error() string {
	return n.message
}

func asNotFound(e error) error {
	return notFound{fmt.Sprintf("Not found: %v", e)}
}

func IsNotFound(err error) bool {
	_, ok := err.(notFound)
	return ok
}

func ExpectOK(res *http.Response, msg string) error {
	return ExpectStatus(res, http.StatusOK, msg)
}

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
