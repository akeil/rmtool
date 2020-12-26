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
	Orientation Orientation
	Template    string
	Size        TemplateSize
	Text        string
}

// HasTemplate tells if the page has a (visible) background template.
func (p *Pagedata) HasTemplate() bool {
	return p.Text != "Blank" && p.Text != ""
}

func (p *Pagedata) Validate() error {
	// TODO implement
	return nil
}

func ReadPagedata(r io.Reader) ([]Pagedata, error) {
	pd := make([]Pagedata, 0)
	s := bufio.NewScanner(r)

	var text string
	var err error
	var size TemplateSize
	var layout Orientation
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

		// Special case: some templates do not have the orientation prefix
		switch text {
		case "Blank",
			"Isometric",
			"Perspective1",
			"Perspective2":
			pd = append(pd, Pagedata{
				Orientation: Portrait,
				Template:    text,
				Size:        TemplateMedium,
				Text:        text,
			})
		default:
			// TODO some templates have no size
			parts = strings.SplitN(text, " ", 3)
			if len(parts) != 3 {
				return pd, fmt.Errorf("invalid pagedata line: %q", text)
			}
			size = size.FromString(parts[2])
			layout = layout.fromString(parts[0])
			pd = append(pd, Pagedata{
				Orientation: layout,
				Template:    parts[1],
				Size:        size,
				Text:        text,
			})
		}
	}

	return pd, nil
}

func (t TemplateSize) FromString(s string) TemplateSize {
	switch s {
	case "S", "small":
		return TemplateSmall
	case "M", "medium", "med":
		return TemplateMedium
	case "L", "large":
		return TemplateLarge
	}
	return TemplateNoSize
}

func (o Orientation) fromString(s string) Orientation {
	switch s {
	case "P":
		return Portrait
	case "LS":
		return Landscape
	default:
		return Portrait
	}
}

func (o Orientation) toString() string {
	switch o {
	case Portrait:
		return "P"
	case Landscape:
		return "LS"
	default:
		return ""
	}
}
