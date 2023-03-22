package route

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Make sure parseUrl is strict enough not to accept total garbage
func Test_rawPath(t *testing.T) {
	values := []struct {
		URL      string
		Expected string
	}{
		{URL: "http://google.com/", Expected: "/"},
		{URL: "http://google.com/a?q=b", Expected: "/a"},
		{URL: "http://google.com/%2Fvalue/hello", Expected: "/%2Fvalue/hello"},
		{URL: "/home", Expected: "/home"},
		{URL: "/home?a=b", Expected: "/home"},
		{URL: "/home%2F", Expected: "/home%2F"},
		{URL: "/oauth/callback?scope=email%20https://www.googleapis.com/auth/userinfo.email%20openid", Expected: "/oauth/callback"},
	}
	for _, v := range values {
		out := rawPath(makeReq(req{url: v.URL}))
		assert.Equal(t, v.Expected, out)
	}
}
