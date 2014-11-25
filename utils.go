package route

import (
	"fmt"
	"net/url"
	"strings"
)

// RawPath returns escaped url path section
func RawPath(in string) (string, error) {
	u, err := url.ParseRequestURI(in)
	if err != nil {
		return "", err
	}
	path := ""
	if u.Opaque != "" {
		path = u.Opaque
	} else if u.Host == "" {
		path = in
	} else {
		vals := strings.SplitN(in, u.Host, 2)
		if len(vals) != 2 {
			return "", fmt.Errorf("failed to parse url")
		}
		path = vals[1]
	}
	idx := strings.IndexRune(path, '?')
	if idx == -1 {
		return path, nil
	}
	return path[:idx], nil
}
