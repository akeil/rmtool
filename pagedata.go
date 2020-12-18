package rm

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type TemplateSize int

const (
	TemplateNoSize TemplateSize = iota
	TemplateSmall
	TemplateMedium
	TemplateLarge
)

type Pagedata struct {
	Prefix   string
	Template string
	Size     TemplateSize
}

func ReadPagedata(r io.Reader) ([]Pagedata, error) {
	pd := make([]Pagedata, 0)
	s := bufio.NewScanner(r)

	var text string
	var err error
	var size TemplateSize
	var parts []string
	for s.Scan() {
		text = s.Text()
		err = s.Err()
		if err != nil {
			return pd, err
		}
		// TODO: assumes that empty lines are allowed - correct?
		if text == "" {
			continue
		}

		// Special case: no template selected
		if text == "Blank" {
			pd = append(pd, Pagedata{Template: "Blank"})
			continue
		}

		parts = strings.SplitN(text, " ", 3)
		if len(parts) != 3 {
			return pd, fmt.Errorf("invalid pagedata line: %q", text)
		}
		size = size.FromString(parts[2])
		pd = append(pd, Pagedata{Prefix: parts[0], Template: parts[1], Size: size})
	}

	return pd, nil
}

func (t TemplateSize) FromString(s string) TemplateSize {
	switch s {
	case "S", "small":
		return TemplateSmall
	case "M", "medium":
		return TemplateMedium
	case "L", "large":
		return TemplateLarge
	}
	return TemplateNoSize
}
