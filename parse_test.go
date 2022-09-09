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
			`Path("/helloworld")`,
			`http://google.com/helloworld`,
			"GET",
			"localhost",
			nil,
		},
		{
			`Method("GET") && Path("/helloworld")`,
			`http://google.com/helloworld`,
			"GET",
			"localhost",
			nil,
		},
		{
			`Path("/hello/<world>")`,
			`http://google.com/hello/world`,
			"GET",
			"localhost",
			nil,
		},
		{
			`Method("POST") &&  Path("/helloworld%2F")`,
			`http://google.com/helloworld%2F`,
			"POST",
			"localhost",
			nil,
		},
		{
			`Method("POST") && Path("/helloworld%2F")`,
			`http://google.com/helloworld%2F?q=b`,
			"POST",
			"localhost",
			nil,
		},
		{
			`Method("POST") && Path("/helloworld/<name>")`,
			`http://google.com/helloworld/%2F`,
			"POST",
			"localhost",
			nil,
		},
		{
			`Method("POST") && Path("/helloworld/<path:name>")`,
			`http://google.com/helloworld/some/name`,
			"POST",
			"localhost",
			nil,
		},
		{
			`Method("POST") && Path("/escaped/<path:name>")`,
			`http://google.com/escaped/some%2Fpath`,
			"POST",
			"localhost",
			nil,
		},
		{
			`Path("/helloworld")`,
			`http://google.com/helloworld`,
			"GET",
			"localhost",
			nil,
		},
		{
			`Method("POST") && Path("/helloworld")`,
			`http://google.com/helloworld`,
			"POST",
			"localhost",
			nil,
		},
		{
			`Host("localhost") && Method("POST") && Path("/helloworld")`,
			`http://google.com/helloworld`,
			"POST",
			"localhost",
			nil,
		},
		{
			`Host("<subdomain>.localhost") && Method("POST") && Path("/helloworld")`,
			`http://google.com/helloworld`,
			"POST",
			"a.localhost",
			nil,
		},
		{
			`Host("<sub1>.<sub2>.localhost") && Method("POST") && Path("/helloworld")`,
			`http://google.com/helloworld`,
			"POST",
			"a.b.localhost",
			nil,
		},
		{
			`Host("<sub1>.<sub2>.localhost") && Method("POST") && Path("/helloworld")`,
			`http://google.com/helloworld`,
			"POST",
			"a.b.localhost",
			nil,
		},
		{
			`Header("Content-Type", "application/json")`,
			`http://google.com/helloworld`,
			"POST",
			"",
			map[string][]string{"Content-Type": []string{"application/json"}},
		},
		{
			`Header("Content-Type", "application/<string>")`,
			`http://google.com/helloworld`,
			"POST",
			"",
			map[string][]string{"Content-Type": []string{"application/json"}},
		},
		{
			`Host("<sub1>.<sub2>.localhost") && Method("POST") && Path("/helloworld") && Header("Content-Type", "application/<string>")`,
			`http://google.com/helloworld`,
			"POST",
			"a.b.localhost",
			map[string][]string{"Content-Type": []string{"application/json"}},
		},
		// Regexp cases
		{
			`PathRegexp("/helloworld")`,
			`http://google.com/helloworld`,
			"GET",
			"localhost",
			nil,
		},
		{
			`HostRegexp("[^\\.]+\\.localhost") && Method("POST") && PathRegexp("/hello.*")`,
			`http://google.com/helloworld`,
			"POST",
			"a.localhost",
			nil,
		},
		{
			`HostRegexp("[^\\.]+\\.localhost") && Method("POST") && PathRegexp("/hello.*") && HeaderRegexp("Content-Type", "application/.+")`,
			`http://google.com/helloworld`,
			"POST",
			"a.b.localhost",
			map[string][]string{"Content-Type": []string{"application/json"}},
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
	testCases := []string{
		`bad`,                             // unsupported identifier
		`bad expression`,                  // not a valid go expression
		`Path("/path") || Path("/path2")`, // unsupported operator
		`1 && 2`,                          // unsupported statements
		`"standalone literal"`,            // standalone literal
		`UnknownFunction("hi")`,           // unknown function
		`Path(1)`,                         // bad argument type
		`RegexpRoute(1)`,                  // bad argument type
		`Path()`,                          // no arguments
		`PathRegexp()`,                    // no arguments
		`Path(Path("hello"))`,             // nested calls
		`Path("")`,                        // bad trie expression
		`PathRegexp("[[[[")`,              // bad regular expression
	}

	for _, expr := range testCases {
		m, err := parse(expr, &match{val: "ok"})
		assert.Error(t, err)
		assert.Nil(t, m)
	}
}
