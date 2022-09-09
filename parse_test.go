package route

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseAndMatchSuccess(t *testing.T) {
	testCases := []struct {
		Expression string
		Url        string
		Method     string
		Host       string
		Headers    http.Header
	}{
		// Trie cases
		{
			Expression: `Path("/helloworld")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodGet,
			Host:       "localhost",
		},
		{
			Expression: `Method("GET") && Path("/helloworld")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodGet,
			Host:       "localhost",
		},
		{
			Expression: `Path("/hello/<world>")`,
			Url:        `http://google.com/hello/world`,
			Method:     http.MethodGet,
			Host:       "localhost",
		},
		{
			Expression: `Method("POST") &&  Path("/helloworld%2F")`,
			Url:        `http://google.com/helloworld%2F`,
			Method:     http.MethodPost,
			Host:       "localhost",
		},
		{
			Expression: `Method("POST") && Path("/helloworld%2F")`,
			Url:        `http://google.com/helloworld%2F?q=b`,
			Method:     http.MethodPost,
			Host:       "localhost",
		},
		{
			Expression: `Method("POST") && Path("/helloworld/<name>")`,
			Url:        `http://google.com/helloworld/%2F`,
			Method:     http.MethodPost,
			Host:       "localhost",
		},
		{
			Expression: `Method("POST") && Path("/helloworld/<path:name>")`,
			Url:        `http://google.com/helloworld/some/name`,
			Method:     http.MethodPost,
			Host:       "localhost",
		},
		{
			Expression: `Method("POST") && Path("/escaped/<path:name>")`,
			Url:        `http://google.com/escaped/some%2Fpath`,
			Method:     http.MethodPost,
			Host:       "localhost",
		},
		{
			Expression: `Path("/helloworld")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodGet,
			Host:       "localhost",
		},
		{
			Expression: `Method("POST") && Path("/helloworld")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Host:       "localhost",
		},
		{
			Expression: `Host("localhost") && Method("POST") && Path("/helloworld")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Host:       "localhost",
		},
		{
			Expression: `Host("<subdomain>.localhost") && Method("POST") && Path("/helloworld")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Host:       "a.localhost",
		},
		{
			Expression: `Host("<sub1>.<sub2>.localhost") && Method("POST") && Path("/helloworld")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Host:       "a.b.localhost",
		},
		{
			Expression: `Host("<sub1>.<sub2>.localhost") && Method("POST") && Path("/helloworld")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Host:       "a.b.localhost",
		},
		{
			Expression: `Header("Content-Type", "application/json")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
		},
		{
			Expression: `Header("Content-Type", "application/<string>")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
		},
		{
			Expression: `Host("<sub1>.<sub2>.localhost") && Method("POST") && Path("/helloworld") && Header("Content-Type", "application/<string>")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Host:       "a.b.localhost",
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
		},
		// Regexp cases
		{
			Expression: `PathRegexp("/helloworld")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodGet,
			Host:       "localhost",
		},
		{
			Expression: `HostRegexp("[^\\.]+\\.localhost") && Method("POST") && PathRegexp("/hello.*")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Host:       "a.localhost",
		},
		{
			Expression: `HostRegexp("[^\\.]+\\.localhost") && Method("POST") && PathRegexp("/hello.*") && HeaderRegexp("Content-Type", "application/.+")`,
			Url:        `http://google.com/helloworld`,
			Method:     http.MethodPost,
			Host:       "a.b.localhost",
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Expression, func(t *testing.T) {
			result := &match{val: "ok"}
			p, err := parse(tc.Expression, result)
			assert.NoError(t, err)
			assert.NotNil(t, p)

			req := makeReq(req{url: tc.Url})
			req.Method = tc.Method
			req.Host = tc.Host
			req.Header = tc.Headers

			out := p.match(req)
			assert.NotNil(t, p)
			assert.Equal(t, result, out)
		})
	}
}

func TestParseFailures(t *testing.T) {
	testCases := []struct {
		desc string
		expr string
	}{
		{
			desc: "unsupported identifier",
			expr: `bad`,
		},
		{
			desc: "not a valid go expression",
			expr: `bad expression`,
		},
		{
			desc: "unsupported operator",
			expr: `Path("/path") || Path("/path2")`,
		},
		{
			desc: "unsupported statements",
			expr: `1 && 2`,
		},
		{
			desc: "standalone literal",
			expr: `"standalone literal"`,
		},
		{
			desc: "unknown function",
			expr: `UnknownFunction("hi")`,
		},
		{
			desc: "bad argument type",
			expr: `Path(1)`,
		},
		{
			desc: "bad argument type",
			expr: `RegexpRoute(1)`,
		},
		{
			desc: "no arguments",
			expr: `Path()`,
		},
		{
			desc: "no arguments",
			expr: `PathRegexp()`,
		},
		{
			desc: "nested calls",
			expr: `Path(Path("hello"))`,
		},
		{
			desc: "bad trie expression",
			expr: `Path("")`,
		},
		{
			desc: "bad regular expression",
			expr: `PathRegexp("[[[[")`,
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.desc, func(t *testing.T) {
			t.Parallel()

			m, err := parse(test.expr, &match{val: "ok"})
			assert.Error(t, err)
			assert.Nil(t, m)
		})
	}
}
