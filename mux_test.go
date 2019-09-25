package route

import (
	"bytes"
	"net/http"

	. "gopkg.in/check.v1"
)

type MuxSuite struct {
}

var _ = Suite(&MuxSuite{})

func (s *MuxSuite) TestEmptyOperationsSucceed(c *C) {
	r := NewMux()

	t := newWriter()
	r.ServeHTTP(t, makeReq(req{url: "/hello"}))

	c.Assert(t.header, Equals, 404)
	c.Assert(t.buf.String(), Equals, "Not found")
}

func (s *MuxSuite) TestRouting(c *C) {
	r := NewMux()

	err := r.HandleFunc(`Host("localhost") && Path("/p")`, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(201)
		w.Write([]byte("/p"))
	})
	c.Assert(err, IsNil)

	t := newWriter()
	r.ServeHTTP(t, makeReq(req{url: "/p", host: "localhost"}))

	c.Assert(t.header, Equals, 201)
	c.Assert(t.buf.String(), Equals, "/p")
}

func (s *MuxSuite) TestInitHandlers(c *C) {
	r := NewMux()

	handlers := map[string]interface{}{
		`Host("localhost") && Path("/p")`: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(201)
			w.Write([]byte("/p"))
		}),
		`Host("localhost") && Path("/f")`: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(201)
			w.Write([]byte("/f"))
		}),
	}

	err := r.InitHandlers(handlers)
	c.Assert(err, IsNil)

	t := newWriter()
	r.ServeHTTP(t, makeReq(req{url: "/p", host: "localhost"}))

	c.Assert(t.header, Equals, 201)
	c.Assert(t.buf.String(), Equals, "/p")

	t = newWriter()
	r.ServeHTTP(t, makeReq(req{url: "/f", host: "localhost"}))

	c.Assert(t.header, Equals, 201)
	c.Assert(t.buf.String(), Equals, "/f")
}

func (s *MuxSuite) TestAddAlias(c *C) {
	expr := `Host("localhost") && Path("/p")`
	f := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(201)
		w.Write([]byte("/p"))
	}
	handler := http.HandlerFunc(f)

	tests := []struct {
		Init func(r *Mux) error
	}{
		{
			func(r *Mux) error {
				return r.InitHandlers(map[string]interface{}{expr: handler})
			},
		},
		{
			func(r *Mux) error {
				return r.Handle(expr, handler)
			},
		},
		{
			func(r *Mux) error {
				return r.HandleFunc(expr, f)
			},
		},
	}

	for _, t := range tests {
		r := NewMux()
		r.AddAlias(`Host("localhost")`, `Host("vulcand.net")`)

		err := t.Init(r)
		c.Assert(err, IsNil)

		// Should not route localhost requests
		w := newWriter()
		r.ServeHTTP(w, makeReq(req{url: "/p", host: "localhost"}))
		c.Assert(w.header, Equals, 404)

		// Should route vulcand.net
		w = newWriter()
		r.ServeHTTP(w, makeReq(req{url: "/p", host: "vulcand.net"}))
		c.Assert(w.header, Equals, 201)
		c.Assert(w.buf.String(), Equals, "/p")

		// After removing the expression
		err = r.Remove(expr)
		c.Assert(err, IsNil)

		// Should NOT route vulcand.net
		w = newWriter()
		r.ServeHTTP(w, makeReq(req{url: "/p", host: "vulcand.net"}))
		c.Assert(w.header, Equals, 404)
	}

}

type testWriter struct {
	header  int
	buf     *bytes.Buffer
	headers http.Header
}

func newWriter() *testWriter {
	return &testWriter{
		buf:     &bytes.Buffer{},
		headers: make(http.Header),
	}
}

func (t *testWriter) Header() http.Header {
	return t.headers
}

func (t *testWriter) Write(p []byte) (n int, err error) {
	return t.buf.Write(p)
}

func (t *testWriter) WriteString(s string) (n int, err error) {
	return t.buf.WriteString(s)
}

func (t *testWriter) WriteHeader(h int) {
	t.header = h
}
