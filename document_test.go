package rmtool

import (
	"testing"
)

func TestNewDocument(t *testing.T) {
	d := NewNotebook("My Document", "")
	err := d.Validate()
	if err != nil {
		t.Log("newly created document is not valid")
		t.Error(err)
	}
}
