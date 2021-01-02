package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func resolve(base, endpoint string) (string, error) {
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	e, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	return b.ResolveReference(e).String(), nil
}

// parseTokenExpiration retrieves the expiration time from the user token.
func parseTokenExpiration(token string) (time.Time, error) {
	var t time.Time

	// split the JWT into its parts (header.payload.signature),
	// we are only interested in the `payload`.
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return t, fmt.Errorf("unexpected number of token segments")
	}

	// decode from base64
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return t, err
	}

	// parse the one field we are interested in
	jwt := struct {
		Exp int64 `json:"exp"`
	}{}
	dec := json.NewDecoder(bytes.NewReader(data))
	err = dec.Decode(&jwt)
	if err != nil {
		return t, err
	}

	return time.Unix(jwt.Exp, 0), nil
}
