package auth

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

func generateRedirectURL(returnType string, query string) (u *url.URL, error error) {
	redirectURL, _, _ := strings.Cut(os.Getenv("CORS_ORIGIN"), ",")
	u, err := url.Parse(redirectURL)
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
