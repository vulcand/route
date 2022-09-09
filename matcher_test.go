package route

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostnameCase(t *testing.T) {
	var matcher1, matcher2 matcher
	var req *http.Request
	var err error

	req, err = http.NewRequest(http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	matcher1, err = hostTrieMatcher("example.com")
	require.NoError(t, err)
	matcher2, err = hostTrieMatcher("Example.Com")
	require.NoError(t, err)

	assert.NotNil(t, matcher1.match(req))
	assert.NotNil(t, matcher2.match(req))

	matcher1, err = hostRegexpMatcher(`.*example.com`)
	require.NoError(t, err)
	matcher2, err = hostRegexpMatcher(`.*Example.Com`)
	require.NoError(t, err)

	assert.NotNil(t, matcher1.match(req))
	assert.NotNil(t, matcher2.match(req))
}
