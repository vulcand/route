package route

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type TrieSuite struct {
	suite.Suite
}

func TestTrieSuite(t *testing.T) {
	suite.Run(t, new(TrieSuite))
}

func (s *TrieSuite) TestParseTrieSuccess() {
	m, r := makeTrie(s.T(), "/", &pathMapper{}, "val")
	s.Equal(r, m.match(makeReq(req{url: "http://google.com"})))
}

func (s *TrieSuite) TestParseTrieFailures() {
	paths := []string{
		"",                       // empty path
		"/<uint8:hi>",            // unsupported matcher
		"/<string:hi:omg:hello>", // unsupported matcher parameters
	}
	for _, p := range paths {
		m, err := newTrieMatcher(p, &pathMapper{}, &match{val: "v1"})
		s.Error(err)
		s.Nil(m)
	}
}

func (s *TrieSuite) testPathToTrie(p, trie string) {
	m, _ := makeTrie(s.T(), p, &pathMapper{}, &match{val: "v"})
	s.Equal(trie, printTrie(m))
}

func (s *TrieSuite) TestPrintTries() {
	// Simple path
	s.testPathToTrie("/a", `
root(0)
 node(0:/)
  match(0:a)
`)

	// Path wit default string parameter
	s.testPathToTrie("/<param1>", `
root(0)
 node(0:/)
  match(0:<string:param1>)
`)

	// Path with trailing parameter
	s.testPathToTrie("/m/<string:param1>", `
root(0)
 node(0:/)
  node(0:m)
   node(0:/)
    match(0:<string:param1>)
`)

	// Path with `path` parameter
	s.testPathToTrie("/m/<path:param1>", `
root(0)
 node(0:/)
  node(0:m)
   node(0:/)
    match(0:<path:param1>)
`)

	// Path with  parameter in the middle
	s.testPathToTrie("/m/<string:param1>/a", `
root(0)
 node(0:/)
  node(0:m)
   node(0:/)
    node(0:<string:param1>)
     node(0:/)
      match(0:a)
`)

	// Path with two parameters
	s.testPathToTrie("/m/<string:param1>/<string:param2>", `
root(0)
 node(0:/)
  node(0:m)
   node(0:/)
    node(0:<string:param1>)
     node(0:/)
      match(0:<string:param2>)
`)

}

func (s *TrieSuite) TestMergeTriesCommonPrefix() {
	t1, l1 := makeTrie(s.T(), "/a", &pathMapper{}, &match{val: "v1"})
	t2, l2 := makeTrie(s.T(), "/b", &pathMapper{}, &match{val: "v2"})

	t3, err := t1.merge(t2)
	s.Require().NoError(err)

	expected := `
root(0)
 node(0:/)
  match(0:a)
  match(0:b)
`
	s.Equal(expected, printTrie(t3.(*trie)))

	s.Equal(l1, t3.match(makeReq(req{url: "http://google.com/a"})))
	s.Equal(l2, t3.match(makeReq(req{url: "http://google.com/b"})))
}

func (s *TrieSuite) TestMergeTriesSubtree() {
	t1, l1 := makeTrie(s.T(), "/aa", &pathMapper{}, &match{val: "v1"})
	t2, l2 := makeTrie(s.T(), "/a", &pathMapper{}, &match{val: "v2"})

	t3, err := t1.merge(t2)
	s.Require().NoError(err)

	expected := `
root(0)
 node(0:/)
  match(0:a)
   match(0:a)
`
	s.Equal(printTrie(t3.(*trie)), expected)

	s.Equal(l1, t3.match(makeReq(req{url: "http://google.com/aa"})))
	s.Equal(l2, t3.match(makeReq(req{url: "http://google.com/a"})))
	s.Nil(t3.match(makeReq(req{url: "http://google.com/b"})))
}

func (s *TrieSuite) TestMergeTriesWithCommonParameter() {
	t1, l1 := makeTrie(s.T(), "/a/<string:name>/b", &pathMapper{}, &match{val: "v1"})
	t2, l2 := makeTrie(s.T(), "/a/<string:name>/c", &pathMapper{}, &match{val: "v2"})

	t3, err := t1.merge(t2)
	s.Require().NoError(err)

	expected := `
root(0)
 node(0:/)
  node(0:a)
   node(0:/)
    node(0:<string:name>)
     node(0:/)
      match(0:b)
      match(0:c)
`
	s.Equal(printTrie(t3.(*trie)), expected)

	s.Equal(t3.match(makeReq(req{url: "http://google.com/a/bla/b"})), l1)
	s.Equal(t3.match(makeReq(req{url: "http://google.com/a/bla/c"})), l2)
	s.Nil(t3.match(makeReq(req{url: "http://google.com/a/"})))
}

func (s *TrieSuite) TestMergeTriesWithDivergedParameter() {
	t1, l1 := makeTrie(s.T(), "/a/<string:name1>/b", &pathMapper{}, &match{val: "v1"})
	t2, l2 := makeTrie(s.T(), "/a/<string:name2>/c", &pathMapper{}, &match{val: "v2"})

	t3, err := t1.merge(t2)
	s.Require().NoError(err)

	expected := `
root(0)
 node(0:/)
  node(0:a)
   node(0:/)
    node(0:<string:name1>)
     node(0:/)
      match(0:b)
    node(0:<string:name2>)
     node(0:/)
      match(0:c)
`
	s.Equal(printTrie(t3.(*trie)), expected)

	s.Equal(l1, t3.match(makeReq(req{url: "http://google.com/a/bla/b"})))
	s.Equal(l2, t3.match(makeReq(req{url: "http://google.com/a/bla/c"})))
	s.Nil(t3.match(makeReq(req{url: "http://google.com/a/"})))
}

func (s *TrieSuite) TestMergeTriesWithSamePath() {
	t1, l1 := makeTrie(s.T(), "/a", &pathMapper{}, &match{val: "v1"})
	t2, _ := makeTrie(s.T(), "/a", &pathMapper{}, &match{val: "v2"})

	t3, err := t1.merge(t2)
	s.Require().NoError(err)

	expected := `
root(0)
 node(0:/)
  match(0:a)
`
	s.Equal(expected, printTrie(t3.(*trie)))
	// The first location will match as it will always go first
	s.Equal(l1, t3.match(makeReq(req{url: "http://google.com/a"})))
}

func (s *TrieSuite) TestMergeAndMatchCases() {
	testCases := []struct {
		trees    []string
		url      string
		expected string
	}{
		// Matching /
		{
			trees:    []string{"/"},
			url:      "http://google.com/",
			expected: "/",
		},
		// Matching / when there's no trailing / in url
		{
			trees:    []string{"/"},
			url:      "http://google.com",
			expected: "/",
		},
		// Choosing the longest path
		{
			trees:    []string{"/v2/domains/", "/v2/domains/domain1"},
			url:      "http://google.com/v2/domains/domain1",
			expected: "/v2/domains/domain1",
		},
		// Named parameters
		{
			trees:    []string{"/v1/domains/<string:name>", "/v2/domains/<string:name>"},
			url:      "http://google.com/v2/domains/domain1",
			expected: "/v2/domains/<string:name>",
		},
		// Int matcher, match
		{
			trees:    []string{"/v<int:version>/domains/<string:name>"},
			url:      "http://google.com/v42/domains/domain1",
			expected: "/v<int:version>/domains/<string:name>",
		},
		// Int matcher, no match
		{
			trees:    []string{"/v<int:version>/domains/<string:name>", "/<string:version>/domains/<string:name>"},
			url:      "http://google.com/v42abc/domains/domain1",
			expected: "/<string:version>/domains/<string:name>",
		},
		// Different combinations of named parameters
		{
			trees:    []string{"/v1/domains/<domain>", "/v2/users/<user>/mailboxes/<mbx>"},
			url:      "http://google.com/v2/users/u1/mailboxes/mbx1",
			expected: "/v2/users/<user>/mailboxes/<mbx>",
		},
		// Something that looks like a pattern, but it's not
		{
			trees:    []string{"/v1/<hello"},
			url:      "http://google.com/v1/<hello",
			expected: "/v1/<hello",
		},
	}
	for _, tc := range testCases {
		t, _ := makeTrie(s.T(), tc.trees[0], &pathMapper{}, tc.trees[0])
		for i, pattern := range tc.trees {
			if i == 0 {
				continue
			}
			t2, _ := makeTrie(s.T(), pattern, &pathMapper{}, pattern)
			out, err := t.merge(t2)
			s.Require().NoError(err)
			t = out.(*trie)
		}
		out := t.match(makeReq(req{url: tc.url}))
		s.Equal(tc.expected, out.val)
	}
}

func (s *TrieSuite) TestChainAndMatchCases() {
	tcs := []struct {
		name     string
		tries    []*trie
		req      *http.Request
		expected string
	}{
		{
			name: "Chain method and path",
			tries: []*trie{
				newTrie(s.T(), http.MethodGet, &methodMapper{}, "v1"),
				newTrie(s.T(), "/v1", &pathMapper{}, "v1"),
			},
			req:      makeReq(req{url: "http://localhost/v1", method: http.MethodGet}),
			expected: "v1",
		},
		{
			name: "Chain hostname, method and path",
			tries: []*trie{
				newTrie(s.T(), "h1", &hostMapper{}, "v0"),
				newTrie(s.T(), http.MethodGet, &methodMapper{}, "v1"),
				newTrie(s.T(), "/v1", &pathMapper{}, "v2"),
			},
			req:      makeReq(req{url: "http://localhost/v1", method: http.MethodGet, host: "h1"}),
			expected: "v2",
		},
	}
	for _, tc := range tcs {
		comment := fmt.Sprintf("%v", tc.name)
		var out *trie
		for _, t := range tc.tries {
			if out == nil {
				out = t
				continue
			}
			m, err := out.chain(t)
			s.Require().NoError(err)
			out = m.(*trie)
		}
		result := out.match(tc.req)
		s.NotNil(result, comment)
		s.Equal(tc.expected, result.val, comment)
	}
}

func BenchmarkMatching(b *testing.B) {
	rndString := NewRndString()

	m, _ := makeTrie(b, rndString.MakePath(20, 10), &pathMapper{}, "v")

	for i := 0; i < 10000; i++ {
		t2, _ := makeTrie(b, rndString.MakePath(20, 10), &pathMapper{}, "v")
		out, err := m.merge(t2)
		require.NoError(b, err)
		m = out.(*trie)
	}

	req := makeReq(req{url: fmt.Sprintf("http://google.com/%s", rndString.MakePath(20, 10))})
	for i := 0; i < b.N; i++ {
		m.match(req)
	}
}

func makeTrie(t testing.TB, expr string, mp requestMapper, val interface{}) (*trie, *match) {
	t.Helper()

	l := &match{
		val: val,
	}
	m, err := newTrieMatcher(expr, mp, l)
	require.NoError(t, err)
	require.NotNil(t, m)
	return m, l
}

func newTrie(t testing.TB, expr string, mp requestMapper, val interface{}) *trie {
	t.Helper()

	m, _ := makeTrie(t, expr, mp, val)
	return m
}

type req struct {
	url     string
	host    string
	headers http.Header
	method  string
}

func makeReq(rq req) *http.Request {
	ur, err := url.ParseRequestURI(rq.url)
	if err != nil {
		panic(err)
	}
	r := &http.Request{
		URL:        ur,
		RequestURI: rq.url,
		Host:       rq.host,
		Header:     rq.headers,
		Method:     rq.method,
	}
	return r
}

type RndString struct {
	src rand.Source
}

func NewRndString() *RndString {
	return &RndString{rand.NewSource(time.Now().UTC().UnixNano())}
}

func (r *RndString) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(r.src.Int63()%26 + 97)
	}
	return len(p), nil
}

func (r *RndString) MakeString(n int) string {
	buffer := &bytes.Buffer{}
	_, _ = io.CopyN(buffer, r, int64(n))
	return buffer.String()
}

func (r *RndString) MakePath(varLen, minLen int) string {
	return fmt.Sprintf("/%s", r.MakeString(rand.Intn(varLen)+minLen))
}
