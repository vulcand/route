package route

import (
	"fmt"
	"net/http"
	"regexp"
	"testing"

	"github.com/stretchr/testify/suite"
)

type RouteSuite struct {
	suite.Suite
}

func TestRouteSuite(t *testing.T) {
	suite.Run(t, new(RouteSuite))
}

func (s *RouteSuite) TestEmptyOperationsSucceed() {
	r := New()

	s.Nil(r.GetRoute("bla"))
	s.Nil(r.RemoveRoute("bla"))

	l, err := r.Route(makeReq(req{url: "http://google.com/blabla"}))
	s.Nil(err)
	s.Nil(l)
}

func (s *RouteSuite) TestCRUD() {
	r := New()

	match := "m"
	rt := `Path("/r1")`
	s.Nil(r.AddRoute(rt, match))
	s.Equal(match, r.GetRoute(rt))
	s.Nil(r.RemoveRoute(rt))
	s.Nil(r.GetRoute(rt))
}

func (s *RouteSuite) TestAddTwiceFails() {
	r := New()

	match := "m"
	rt := `Path("/r1")`
	s.Nil(r.AddRoute(rt, match))
	s.NotNil(r.AddRoute(rt, match))

	// Make sure that error did not have side effects
	out, err := r.Route(makeReq(req{url: "http://google.com/r1"}))
	s.Nil(err)
	s.Equal(match, out)
}

func (s *RouteSuite) TestBadExpression() {
	r := New()

	m := "m"
	s.Nil(r.AddRoute(`Path("/r1")`, m))
	s.NotNil(r.AddRoute(`blabla`, "other"))

	// Make sure that error did not have side effects
	out, err := r.Route(makeReq(req{url: "http://google.com/r1"}))
	s.Nil(err)
	s.Equal(m, out)
}

func (s *RouteSuite) TestUpsert() {
	r := New()

	m1, m2 := "m1", "m2"
	s.Nil(r.UpsertRoute(`Path("/r1")`, m1))
	s.Nil(r.UpsertRoute(`Path("/r1")`, m2))
	s.NotNil(r.UpsertRoute(`Path"/r1")`, m2))

	out, err := r.Route(makeReq(req{url: "http://google.com/r1"}))
	s.Nil(err)
	s.Equal(m2, out)
}

func (s *RouteSuite) TestMatchCases() {
	tc := []struct {
		name     string
		routes   []route // routes to add
		tries    []try   // various requests and outcomes
		expected int     // expected compiled matchers
	}{
		{
			name: "Simple Trie Path Matching",
			routes: []route{
				{expr: `Path("/r1")`, match: "m1"},
				{expr: `Path("/r2")`, match: "m2"},
			},
			expected: 1,
			tries: []try{
				{
					r:     req{url: "http://google.com/r1"},
					match: "m1",
				},
				{
					r:     req{url: "http://google.com/r2"},
					match: "m2",
				},
				{
					r: req{url: "http://google.com/r3"},
				},
			},
		},
		{
			name: "Simple Trie Path Matching",
			routes: []route{
				{expr: `Path("/r1")`, match: "m1"},
			},
			expected: 1,
			tries: []try{
				{
					r: req{url: "http://google.com/r3"},
				},
			},
		},
		{
			name: "Regexp path matching",
			routes: []route{
				{expr: `PathRegexp("/r1")`, match: "m1"},
				{expr: `PathRegexp("/r2")`, match: "m2"},
			},
			expected: 2, // Note that router does not compress regular expressions
			tries: []try{
				{
					r:     req{url: "http://google.com/r1"},
					match: "m1",
				},
				{
					r:     req{url: "http://google.com/r2"},
					match: "m2",
				},
				{
					r: req{url: "http://google.com/r3"},
				},
			},
		},
		{
			name: "Mixed matching with trie and regexp",
			routes: []route{
				{expr: `PathRegexp("/r1")`, match: "m1"},
				{expr: `Path("/r2")`, match: "m2"},
			},
			expected: 2, // Note that router does not compress regular expressions
			tries: []try{
				{
					r:     req{url: "http://google.com/r1"},
					match: "m1",
				},
				{
					r:     req{url: "http://google.com/r2"},
					match: "m2",
				},
				{
					r: req{url: "http://google.com/r3"},
				},
			},
		},
		{
			name: "Make sure longest path matches",
			routes: []route{
				{expr: `Path("/r")`, match: "m1"},
				{expr: `Path("/r/hello")`, match: "m2"},
			},
			expected: 1,
			tries: []try{
				{
					r:     req{url: "http://google.com/r/hello"},
					match: "m2",
				},
			},
		},
		{
			name: "Match by method and path",
			routes: []route{
				{expr: `Method("POST") && Path("/r1")`, match: "m1"},
				{expr: `Method("GET") && Path("/r1")`, match: "m2"},
			},
			expected: 1,
			tries: []try{
				{
					r:     req{url: "http://google.com/r1", method: http.MethodPost},
					match: "m1",
				},
				{
					r:     req{url: "http://google.com/r1", method: http.MethodGet},
					match: "m2",
				},
				{
					r: req{url: "http://google.com/r1", method: http.MethodPut},
				},
			},
		},
		{
			name: "Match by method and path",
			routes: []route{
				{expr: `Method("GET") && Path("/v1")`, match: "m1"},
				{expr: `Method("GET") && Path("/v2")`, match: "m2"},
				{expr: `Method("GET") && Path("/v3")`, match: "m3"},
			},
			expected: 1,
			tries: []try{
				{
					r:     req{url: "http://google.com/v1", method: http.MethodGet},
					match: "m1",
				},
				{
					r:     req{url: "http://google.com/v2", method: http.MethodGet},
					match: "m2",
				},
				{
					r:     req{url: "http://google.com/v3", method: http.MethodGet},
					match: "m3",
				},
			},
		},
		{
			name: "Match by method, path and hostname, same method and path",
			routes: []route{
				{expr: `Host("h1") && Method("POST") && Path("/r1")`, match: "m1"},
				{expr: `Host("h2") && Method("POST") && Path("/r1")`, match: "m2"},
			},
			expected: 1,
			tries: []try{
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1"},
					match: "m1",
				},
				{
					r:     req{url: "http://h2/r1", method: http.MethodPost, host: "h2"},
					match: "m2",
				},
				{
					r: req{url: "http://h2/r1", method: http.MethodGet, host: "h2"},
				},
				{
					r: req{url: "http://h2/r1", method: http.MethodGet},
				},
			},
		},
		{
			name: "Match by method, path and hostname, same method and path",
			routes: []route{
				{expr: `Host("h1") && Method("POST") && Path("/r1")`, match: "m1"},
				{expr: `Host("h2") && Method("GET") && Path("/r1")`, match: "m2"},
			},
			expected: 1,
			tries: []try{
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1"},
					match: "m1",
				},
				{
					r:     req{url: "http://h2/r1", method: http.MethodGet, host: "h2"},
					match: "m2",
				},
				{
					r: req{url: "http://h2/r1", method: http.MethodGet},
				},
			},
		},
		{
			name: "Mixed match by method, path and hostname, same method and path",
			routes: []route{
				{expr: `Host("h1") && Method("POST") && Path("/r1")`, match: "m1"},
				{expr: `HostRegexp("h2") && Method("POST") && Path("/r1")`, match: "m2"},
			},
			expected: 2,
			tries: []try{
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1"},
					match: "m1",
				},
				{
					r:     req{url: "http://h2/r1", method: http.MethodPost, host: "h2"},
					match: "m2",
				},
			},
		},
		{
			name: "Match by regexp method",
			routes: []route{
				{expr: `MethodRegexp("POST|PUT") && Path("/r1")`, match: "m1"},
				{expr: `MethodRegexp("GET") && Path("/r1")`, match: "m2"},
			},
			expected: 2,
			tries: []try{
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost},
					match: "m1",
				},
				{
					r:     req{url: "http://h1/r1", method: http.MethodPut},
					match: "m1",
				},
				{
					r:     req{url: "http://h2/r1", method: http.MethodGet},
					match: "m2",
				},
			},
		},
		{
			name: "Match by method, path and hostname and header",
			routes: []route{
				{expr: `Host("h1") && Method("POST") && Path("/r1")`, match: "m1"},
				{expr: `Host("h2") && Method("POST") && Path("/r1") && Header("Content-Type", "application/json")`, match: "m2"},
			},
			expected: 1,
			tries: []try{
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1"},
					match: "m1",
				},
				{
					r:     req{url: "http://h2/r1", method: http.MethodPost, host: "h2", headers: http.Header{"Content-Type": []string{"application/json"}}},
					match: "m2",
				},
			},
		},
		{
			name: "Match by method, path and hostname and header for same hosts",
			routes: []route{
				{expr: `Host("h1") && Method("POST") && Path("/r1")`, match: "m1"},
				{expr: `Host("h1") && Method("POST") && Path("/r1") && Header("Content-Type", "application/json")`, match: "m2"},
			},
			expected: 1,
			tries: []try{
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1"},
					match: "m1",
				},
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1", headers: http.Header{"Content-Type": []string{"application/json"}}},
					match: "m2",
				},
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1", headers: http.Header{"Content-Type": []string{"text/plain"}}},
					match: "m1",
				},
			},
		},
		{
			name: "Catch all match for content-type",
			routes: []route{
				{expr: `Host("h1") && Method("POST") && Path("/r1") && Header("Content-Type", "<string>/<string>")`, match: "m1"},
				{expr: `Host("h1") && Method("POST") && Path("/r1") && Header("Content-Type", "application/json")`, match: "m2"},
			},
			expected: 1,
			tries: []try{
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1", headers: http.Header{"Content-Type": []string{"text/plain"}}},
					match: "m1",
				},
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1", headers: http.Header{"Content-Type": []string{"application/json"}}},
					match: "m2",
				},
			},
		},
		{
			name: "Match by method, path and hostname and header regexp",
			routes: []route{
				{expr: `Host("h1") && Method("POST") && Path("/r1")`, match: "m1"},
				{expr: `Host("h2") && Method("POST") && Path("/r1") && HeaderRegexp("Content-Type", "application/.*")`, match: "m2"},
			},
			expected: 2,
			tries: []try{
				{
					r:     req{url: "http://h1/r1", method: http.MethodPost, host: "h1"},
					match: "m1",
				},
				{
					r:     req{url: "http://h2/r1", method: http.MethodPost, host: "h2", headers: http.Header{"Content-Type": []string{"application/json"}}},
					match: "m2",
				},
			},
		},
		{
			name: "Make sure there is no match overlap",
			routes: []route{
				{expr: `Host("h1") && Method("POST") && Path("/r1")`, match: "m1"},
			},
			expected: 1,
			tries: []try{
				{
					r: req{url: "http://h/r1", method: "1POST", host: "h"},
				},
			},
		},
	}
	for _, test := range tc {
		comment := fmt.Sprintf("%v", test.name)

		r := New().(*router)
		for _, rt := range test.routes {
			s.Nil(r.AddRoute(rt.expr, rt.match), comment)
		}
		if test.expected != 0 {
			s.Len(r.matchers, test.expected, comment)
		}

		for _, a := range test.tries {
			req := makeReq(a.r)

			out, err := r.Route(req)
			s.Nil(err)
			if a.match != "" {
				s.Equal(a.match, out, comment)
			} else {
				s.Nil(out, comment)
			}
		}
	}
}

func (s *RouteSuite) TestGithubAPI() {
	r := New()

	re := regexp.MustCompile(":([^/]*)")
	for _, sp := range githubAPI {
		path := re.ReplaceAllString(sp.path, "<$1>")
		expr := fmt.Sprintf(`Method("%s") && Path("%s")`, sp.method, path)
		s.Nil(r.AddRoute(expr, expr))
	}

	for _, sp := range githubAPI {
		path := re.ReplaceAllString(sp.path, "<$1>")
		expr := fmt.Sprintf(`Method("%s") && Path("%s")`, sp.method, path)
		out, err := r.Route(makeReq(req{method: sp.method, url: sp.path}))
		s.Nil(err)
		s.Equal(expr, out)
	}
}

type route struct {
	expr  string
	match string
}

type try struct {
	r     req
	match string
}

type spec struct {
	method string
	path   string
}

var githubAPI = []spec{
	// OAuth Authorizations
	{http.MethodGet, "/authorizations"},
	{http.MethodGet, "/authorizations/:id"},
	{http.MethodPost, "/authorizations"},
	// {http.MethodPut, "/authorizations/clients/:client_id"},
	// {http.MethodPatch, "/authorizations/:id"},
	{http.MethodDelete, "/authorizations/:id"},
	{http.MethodGet, "/applications/:client_id/tokens/:access_token"},
	{http.MethodDelete, "/applications/:client_id/tokens"},
	{http.MethodDelete, "/applications/:client_id/tokens/:access_token"},

	// Activity
	{http.MethodGet, "/events"},
	{http.MethodGet, "/repos/:owner/:repo/events"},
	{http.MethodGet, "/networks/:owner/:repo/events"},
	{http.MethodGet, "/orgs/:org/events"},
	{http.MethodGet, "/users/:user/received_events"},
	{http.MethodGet, "/users/:user/received_events/public"},
	{http.MethodGet, "/users/:user/events"},
	{http.MethodGet, "/users/:user/events/public"},
	{http.MethodGet, "/users/:user/events/orgs/:org"},
	{http.MethodGet, "/feeds"},
	{http.MethodGet, "/notifications"},
	{http.MethodGet, "/repos/:owner/:repo/notifications"},
	{http.MethodPut, "/notifications"},
	{http.MethodPut, "/repos/:owner/:repo/notifications"},
	{http.MethodGet, "/notifications/threads/:id"},
	// {http.MethodPatch, "/notifications/threads/:id"},
	{http.MethodGet, "/notifications/threads/:id/subscription"},
	{http.MethodPut, "/notifications/threads/:id/subscription"},
	{http.MethodDelete, "/notifications/threads/:id/subscription"},
	{http.MethodGet, "/repos/:owner/:repo/stargazers"},
	{http.MethodGet, "/users/:user/starred"},
	{http.MethodGet, "/user/starred"},
	{http.MethodGet, "/user/starred/:owner/:repo"},
	{http.MethodPut, "/user/starred/:owner/:repo"},
	{http.MethodDelete, "/user/starred/:owner/:repo"},
	{http.MethodGet, "/repos/:owner/:repo/subscribers"},
	{http.MethodGet, "/users/:user/subscriptions"},
	{http.MethodGet, "/user/subscriptions"},
	{http.MethodGet, "/repos/:owner/:repo/subscription"},
	{http.MethodPut, "/repos/:owner/:repo/subscription"},
	{http.MethodDelete, "/repos/:owner/:repo/subscription"},
	{http.MethodGet, "/user/subscriptions/:owner/:repo"},
	{http.MethodPut, "/user/subscriptions/:owner/:repo"},
	{http.MethodDelete, "/user/subscriptions/:owner/:repo"},

	// Gists
	{http.MethodGet, "/users/:user/gists"},
	{http.MethodGet, "/gists"},
	// {http.MethodGet, "/gists/public"},
	// {http.MethodGet, "/gists/starred"},
	{http.MethodGet, "/gists/:id"},
	{http.MethodPost, "/gists"},
	// {http.MethodPatch, "/gists/:id"},
	{http.MethodPut, "/gists/:id/star"},
	{http.MethodDelete, "/gists/:id/star"},
	{http.MethodGet, "/gists/:id/star"},
	{http.MethodPost, "/gists/:id/forks"},
	{http.MethodDelete, "/gists/:id"},

	// Git Data
	{http.MethodGet, "/repos/:owner/:repo/git/blobs/:sha"},
	{http.MethodPost, "/repos/:owner/:repo/git/blobs"},
	{http.MethodGet, "/repos/:owner/:repo/git/commits/:sha"},
	{http.MethodPost, "/repos/:owner/:repo/git/commits"},
	// {http.MethodGet, "/repos/:owner/:repo/git/refs/*ref"},
	{http.MethodGet, "/repos/:owner/:repo/git/refs"},
	{http.MethodPost, "/repos/:owner/:repo/git/refs"},
	// {http.MethodPatch, "/repos/:owner/:repo/git/refs/*ref"},
	// {http.MethodDelete, "/repos/:owner/:repo/git/refs/*ref"},
	{http.MethodGet, "/repos/:owner/:repo/git/tags/:sha"},
	{http.MethodPost, "/repos/:owner/:repo/git/tags"},
	{http.MethodGet, "/repos/:owner/:repo/git/trees/:sha"},
	{http.MethodPost, "/repos/:owner/:repo/git/trees"},

	// Issues
	{http.MethodGet, "/issues"},
	{http.MethodGet, "/user/issues"},
	{http.MethodGet, "/orgs/:org/issues"},
	{http.MethodGet, "/repos/:owner/:repo/issues"},
	{http.MethodGet, "/repos/:owner/:repo/issues/:number"},
	{http.MethodPost, "/repos/:owner/:repo/issues"},
	// {http.MethodPatch, "/repos/:owner/:repo/issues/:number"},
	{http.MethodGet, "/repos/:owner/:repo/assignees"},
	{http.MethodGet, "/repos/:owner/:repo/assignees/:assignee"},
	{http.MethodGet, "/repos/:owner/:repo/issues/:number/comments"},
	// {http.MethodGet, "/repos/:owner/:repo/issues/comments"},
	// {http.MethodGet, "/repos/:owner/:repo/issues/comments/:id"},
	{http.MethodPost, "/repos/:owner/:repo/issues/:number/comments"},
	// {http.MethodPatch, "/repos/:owner/:repo/issues/comments/:id"},
	// {http.MethodDelete, "/repos/:owner/:repo/issues/comments/:id"},
	{http.MethodGet, "/repos/:owner/:repo/issues/:number/events"},
	// {http.MethodGet, "/repos/:owner/:repo/issues/events"},
	// {http.MethodGet, "/repos/:owner/:repo/issues/events/:id"},
	{http.MethodGet, "/repos/:owner/:repo/labels"},
	{http.MethodGet, "/repos/:owner/:repo/labels/:name"},
	{http.MethodPost, "/repos/:owner/:repo/labels"},
	// {http.MethodPatch, "/repos/:owner/:repo/labels/:name"},
	{http.MethodDelete, "/repos/:owner/:repo/labels/:name"},
	{http.MethodGet, "/repos/:owner/:repo/issues/:number/labels"},
	{http.MethodPost, "/repos/:owner/:repo/issues/:number/labels"},
	{http.MethodDelete, "/repos/:owner/:repo/issues/:number/labels/:name"},
	{http.MethodPut, "/repos/:owner/:repo/issues/:number/labels"},
	{http.MethodDelete, "/repos/:owner/:repo/issues/:number/labels"},
	{http.MethodGet, "/repos/:owner/:repo/milestones/:number/labels"},
	{http.MethodGet, "/repos/:owner/:repo/milestones"},
	{http.MethodGet, "/repos/:owner/:repo/milestones/:number"},
	{http.MethodPost, "/repos/:owner/:repo/milestones"},
	// {http.MethodPatch, "/repos/:owner/:repo/milestones/:number"},
	{http.MethodDelete, "/repos/:owner/:repo/milestones/:number"},

	// Miscellaneous
	{http.MethodGet, "/emojis"},
	{http.MethodGet, "/gitignore/templates"},
	{http.MethodGet, "/gitignore/templates/:name"},
	{http.MethodPost, "/markdown"},
	{http.MethodPost, "/markdown/raw"},
	{http.MethodGet, "/meta"},
	{http.MethodGet, "/rate_limit"},

	// Organizations
	{http.MethodGet, "/users/:user/orgs"},
	{http.MethodGet, "/user/orgs"},
	{http.MethodGet, "/orgs/:org"},
	// {http.MethodPatch, "/orgs/:org"},
	{http.MethodGet, "/orgs/:org/members"},
	{http.MethodGet, "/orgs/:org/members/:user"},
	{http.MethodDelete, "/orgs/:org/members/:user"},
	{http.MethodGet, "/orgs/:org/public_members"},
	{http.MethodGet, "/orgs/:org/public_members/:user"},
	{http.MethodPut, "/orgs/:org/public_members/:user"},
	{http.MethodDelete, "/orgs/:org/public_members/:user"},
	{http.MethodGet, "/orgs/:org/teams"},
	{http.MethodGet, "/teams/:id"},
	{http.MethodPost, "/orgs/:org/teams"},
	// {http.MethodPatch, "/teams/:id"},
	{http.MethodDelete, "/teams/:id"},
	{http.MethodGet, "/teams/:id/members"},
	{http.MethodGet, "/teams/:id/members/:user"},
	{http.MethodPut, "/teams/:id/members/:user"},
	{http.MethodDelete, "/teams/:id/members/:user"},
	{http.MethodGet, "/teams/:id/repos"},
	{http.MethodGet, "/teams/:id/repos/:owner/:repo"},
	{http.MethodPut, "/teams/:id/repos/:owner/:repo"},
	{http.MethodDelete, "/teams/:id/repos/:owner/:repo"},
	{http.MethodGet, "/user/teams"},

	// Pull Requests
	{http.MethodGet, "/repos/:owner/:repo/pulls"},
	{http.MethodGet, "/repos/:owner/:repo/pulls/:number"},
	{http.MethodPost, "/repos/:owner/:repo/pulls"},
	// {http.MethodPatch, "/repos/:owner/:repo/pulls/:number"},
	{http.MethodGet, "/repos/:owner/:repo/pulls/:number/commits"},
	{http.MethodGet, "/repos/:owner/:repo/pulls/:number/files"},
	{http.MethodGet, "/repos/:owner/:repo/pulls/:number/merge"},
	{http.MethodPut, "/repos/:owner/:repo/pulls/:number/merge"},
	{http.MethodGet, "/repos/:owner/:repo/pulls/:number/comments"},
	// {http.MethodGet, "/repos/:owner/:repo/pulls/comments"},
	// {http.MethodGet, "/repos/:owner/:repo/pulls/comments/:number"},
	{http.MethodPut, "/repos/:owner/:repo/pulls/:number/comments"},
	// {http.MethodPatch, "/repos/:owner/:repo/pulls/comments/:number"},
	// {http.MethodDelete, "/repos/:owner/:repo/pulls/comments/:number"},

	// Repositories
	{http.MethodGet, "/user/repos"},
	{http.MethodGet, "/users/:user/repos"},
	{http.MethodGet, "/orgs/:org/repos"},
	{http.MethodGet, "/repositories"},
	{http.MethodPost, "/user/repos"},
	{http.MethodPost, "/orgs/:org/repos"},
	{http.MethodGet, "/repos/:owner/:repo"},
	// {http.MethodPatch, "/repos/:owner/:repo"},
	{http.MethodGet, "/repos/:owner/:repo/contributors"},
	{http.MethodGet, "/repos/:owner/:repo/languages"},
	{http.MethodGet, "/repos/:owner/:repo/teams"},
	{http.MethodGet, "/repos/:owner/:repo/tags"},
	{http.MethodGet, "/repos/:owner/:repo/branches"},
	{http.MethodGet, "/repos/:owner/:repo/branches/:branch"},
	{http.MethodDelete, "/repos/:owner/:repo"},
	{http.MethodGet, "/repos/:owner/:repo/collaborators"},
	{http.MethodGet, "/repos/:owner/:repo/collaborators/:user"},
	{http.MethodPut, "/repos/:owner/:repo/collaborators/:user"},
	{http.MethodDelete, "/repos/:owner/:repo/collaborators/:user"},
	{http.MethodGet, "/repos/:owner/:repo/comments"},
	{http.MethodGet, "/repos/:owner/:repo/commits/:sha/comments"},
	{http.MethodPost, "/repos/:owner/:repo/commits/:sha/comments"},
	{http.MethodGet, "/repos/:owner/:repo/comments/:id"},
	// {http.MethodPatch, "/repos/:owner/:repo/comments/:id"},
	{http.MethodDelete, "/repos/:owner/:repo/comments/:id"},
	{http.MethodGet, "/repos/:owner/:repo/commits"},
	{http.MethodGet, "/repos/:owner/:repo/commits/:sha"},
	{http.MethodGet, "/repos/:owner/:repo/readme"},
	// {http.MethodGet, "/repos/:owner/:repo/contents/*path"},
	// {http.MethodPut, "/repos/:owner/:repo/contents/*path"},
	// {http.MethodDelete, "/repos/:owner/:repo/contents/*path"},
	// {http.MethodGet, "/repos/:owner/:repo/:archive_format/:ref"},
	{http.MethodGet, "/repos/:owner/:repo/keys"},
	{http.MethodGet, "/repos/:owner/:repo/keys/:id"},
	{http.MethodPost, "/repos/:owner/:repo/keys"},
	// {http.MethodPatch, "/repos/:owner/:repo/keys/:id"},
	{http.MethodDelete, "/repos/:owner/:repo/keys/:id"},
	{http.MethodGet, "/repos/:owner/:repo/downloads"},
	{http.MethodGet, "/repos/:owner/:repo/downloads/:id"},
	{http.MethodDelete, "/repos/:owner/:repo/downloads/:id"},
	{http.MethodGet, "/repos/:owner/:repo/forks"},
	{http.MethodPost, "/repos/:owner/:repo/forks"},
	{http.MethodGet, "/repos/:owner/:repo/hooks"},
	{http.MethodGet, "/repos/:owner/:repo/hooks/:id"},
	{http.MethodPost, "/repos/:owner/:repo/hooks"},
	// {http.MethodPatch, "/repos/:owner/:repo/hooks/:id"},
	{http.MethodPost, "/repos/:owner/:repo/hooks/:id/tests"},
	{http.MethodDelete, "/repos/:owner/:repo/hooks/:id"},
	{http.MethodPost, "/repos/:owner/:repo/merges"},
	{http.MethodGet, "/repos/:owner/:repo/releases"},
	{http.MethodGet, "/repos/:owner/:repo/releases/:id"},
	{http.MethodPost, "/repos/:owner/:repo/releases"},
	// {http.MethodPatch, "/repos/:owner/:repo/releases/:id"},
	{http.MethodDelete, "/repos/:owner/:repo/releases/:id"},
	{http.MethodGet, "/repos/:owner/:repo/releases/:id/assets"},
	{http.MethodGet, "/repos/:owner/:repo/stats/contributors"},
	{http.MethodGet, "/repos/:owner/:repo/stats/commit_activity"},
	{http.MethodGet, "/repos/:owner/:repo/stats/code_frequency"},
	{http.MethodGet, "/repos/:owner/:repo/stats/participation"},
	{http.MethodGet, "/repos/:owner/:repo/stats/punch_card"},
	{http.MethodGet, "/repos/:owner/:repo/statuses/:ref"},
	{http.MethodPost, "/repos/:owner/:repo/statuses/:ref"},

	// Search
	{http.MethodGet, "/search/repositories"},
	{http.MethodGet, "/search/code"},
	{http.MethodGet, "/search/issues"},
	{http.MethodGet, "/search/users"},
	{http.MethodGet, "/legacy/issues/search/:owner/:repository/:state/:keyword"},
	{http.MethodGet, "/legacy/repos/search/:keyword"},
	{http.MethodGet, "/legacy/user/search/:keyword"},
	{http.MethodGet, "/legacy/user/email/:email"},

	// Users
	{http.MethodGet, "/users/:user"},
	{http.MethodGet, "/user"},
	// {http.MethodPatch, "/user"},
	{http.MethodGet, "/users"},
	{http.MethodGet, "/user/emails"},
	{http.MethodPost, "/user/emails"},
	{http.MethodDelete, "/user/emails"},
	{http.MethodGet, "/users/:user/followers"},
	{http.MethodGet, "/user/followers"},
	{http.MethodGet, "/users/:user/following"},
	{http.MethodGet, "/user/following"},
	{http.MethodGet, "/user/following/:user"},
	{http.MethodGet, "/users/:user/following/:target_user"},
	{http.MethodPut, "/user/following/:user"},
	{http.MethodDelete, "/user/following/:user"},
	{http.MethodGet, "/users/:user/keys"},
	{http.MethodGet, "/user/keys"},
	{http.MethodGet, "/user/keys/:id"},
	{http.MethodPost, "/user/keys"},
	// {http.MethodPatch, "/user/keys/:id"},
	{http.MethodDelete, "/user/keys/:id"},
}
