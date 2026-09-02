package auth

import (
	"fmt"
	"net/url"
)

func generateRedirectURL(returnOrigin string, returnType string, query string) (u *url.URL, error error) {
	u, err := url.Parse(returnOrigin)
	if err != nil {
		return nil, fmt.Errorf("error parsing redirectURL: %v", err)
	}
	u = u.JoinPath(returnType)

	if query != "" {
		q := u.Query()
		q.Set("error", query)
		u.RawQuery = q.Encode()
	}
	return u, nil
}
