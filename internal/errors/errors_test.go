package errors

import (
	e "errors"
	"testing"
)

func TestIsNotFound(t *testing.T) {
	err := e.New("some error")
	if IsNotFound(err) {
		t.Log("custom error type NotFound is wrongly recognized")
		t.Fail()
	}

	err = asNotFound(err)
	if !IsNotFound(err) {
		t.Log("custom error type NotFound is not recognized")
		t.Fail()
	}
}
