package rm

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

type Pagedata struct {
	Prefix   string
	Template string
}

func ReadPagedata(r io.Reader) ([]Pagedata, error) {
	pd := make([]Pagedata, 0)
	s := bufio.NewScanner(r)

	var text string
	var err error
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

		parts = strings.SplitN(text, " ", 2)
		if len(parts) != 2 {
			return pd, fmt.Errorf("invalid pagedata line: %q", text)
		}
		pd = append(pd, Pagedata{Prefix: parts[0], Template: parts[1]})
	}

	return pd, nil
}
