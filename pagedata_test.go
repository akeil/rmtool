package rm

import (
	"strings"
	"testing"
)

func TestReadPagedata(t *testing.T) {
	s := "P Lines medium\nP Lines medium\nP Lines medium"
	r := strings.NewReader(s)

	pd, err := ReadPagedata(r)
	if err != nil {
		t.Fatal(err)
	}

	if len(pd) != 3 {
		t.Errorf("Unexpected number of pagedata entries")
	}

	if pd[1].Prefix != "P" {
		t.Errorf("unexpected prefix: %q", pd[1].Prefix)
	}

	if pd[1].Template != "Lines" {
		t.Errorf("unexpected template: %q", pd[1].Template)
	}

	if pd[1].Size != TemplateMedium {
		t.Errorf("unexpected size: %q", pd[1].Size)
	}
}

func TestReadPagedataBlank(t *testing.T) {
	s := "Blank\nBlank"
	r := strings.NewReader(s)

	pd, err := ReadPagedata(r)
	if err != nil {
		t.Fatal(err)
	}

	if len(pd) != 2 {
		t.Errorf("Unexpected number of pagedata entries")
	}

	if pd[1].Prefix != "" {
		t.Errorf("unexpected prefix: %q", pd[1].Prefix)
	}

	if pd[1].Template != "Blank" {
		t.Errorf("unexpected template: %q", pd[1].Template)
	}

	if pd[1].Size != TemplateNoSize {
		t.Errorf("unexpected size: %q", pd[1].Size)
	}
}
