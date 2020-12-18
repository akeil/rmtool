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

type PageLayout int

const (
	Portrait PageLayout = iota
	Landscape
)

type Pagedata struct {
	Layout   PageLayout
	Template string
	Size     TemplateSize
}

func ReadPagedata(r io.Reader) ([]Pagedata, error) {
	pd := make([]Pagedata, 0)
	s := bufio.NewScanner(r)

	var text string
	var err error
	var size TemplateSize
	var layout PageLayout
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
				Layout:   Portrait,
				Template: text,
				Size:     TemplateMedium,
			})
		default:
			// TODO some templates have no size
			parts = strings.SplitN(text, " ", 3)
			if len(parts) != 3 {
				return pd, fmt.Errorf("invalid pagedata line: %q", text)
			}
			size = size.FromString(parts[2])
			layout = layout.FromString(parts[0])
			pd = append(pd, Pagedata{Layout: layout, Template: parts[1], Size: size})
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

func (p PageLayout) FromString(s string) PageLayout {
	switch s {
	case "P":
		return Portrait
	case "LS":
		return Landscape
	default:
		return Portrait
	}
}
