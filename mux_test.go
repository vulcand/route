package route

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

type MuxSuite struct {
	suite.Suite
}

func TestMuxSuite(t *testing.T) {
	suite.Run(t, new(MuxSuite))
}

func (s *MuxSuite) TestEmptyOperationsSucceed() {
	r := NewMux()

	w := newWriter()
	r.ServeHTTP(w, makeReq(req{url: "/hello"}))

	s.Equal(http.StatusNotFound, w.header)
	s.Equal(http.StatusText(http.StatusNotFound), w.buf.String())
}

func (s *MuxSuite) TestRouting() {
	r := NewMux()

	err := r.HandleFunc(`Host("localhost") && Path("/p")`, func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(201)
		_, _ = w.Write([]byte("/p"))
	})
	s.Require().NoError(err)

	w := newWriter()
	r.ServeHTTP(w, makeReq(req{url: "/p", host: "localhost"}))

	s.Equal(http.StatusCreated, w.header)
	s.Equal("/p", w.buf.String())
}

func (s *MuxSuite) TestInitHandlers() {
	r := NewMux()

	handlers := map[string]interface{}{
		`Host("localhost") && Path("/p")`: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("/p"))
		}),
		`Host("localhost") && Path("/f")`: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("/f"))
		}),
	}

	err := r.InitHandlers(handlers)
	s.Require().NoError(err)

	w := newWriter()
	r.ServeHTTP(w, makeReq(req{url: "/p", host: "localhost"}))

	s.Equal(http.StatusCreated, w.header)
	s.Equal("/p", w.buf.String())

	w = newWriter()
	r.ServeHTTP(w, makeReq(req{url: "/f", host: "localhost"}))

	s.Equal(http.StatusCreated, w.header)
	s.Equal("/f", w.buf.String())
}

func (s *MuxSuite) TestAddAlias() {
	expr := `Host("localhost") && Path("/p")`
	f := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("/p"))
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

	for _, test := range tests {
		r := NewMux()
		r.AddAlias(`Host("localhost")`, `Host("vulcand.net")`)
		r.AddAlias(`Path("/p")`, `Path("/g")`)

		err := test.Init(r)
		s.Require().NoError(err)

		// Should continue to route non aliased requests
		w := newWriter()
		r.ServeHTTP(w, makeReq(req{url: "/p", host: "localhost"}))
		s.Equal(http.StatusCreated, w.header)

		// Should route requests to /g for vulcand.net
		r.ServeHTTP(w, makeReq(req{url: "/g", host: "vulcand.net"}))
		s.Equal(http.StatusCreated, w.header)

		// After removing the expression
		err = r.Remove(expr)
		s.Require().NoError(err)

		// Should NOT route /g vulcand.net
		w = newWriter()
		r.ServeHTTP(w, makeReq(req{url: "/g", host: "vulcand.net"}))
		s.Equal(http.StatusNotFound, w.header)
	}
}

func (s *MuxSuite) TestAddAliasOrder() {
	f := func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("/p"))
	}

	r := NewMux()
	// The order in which alases are added can clobber previous alases
	r.AddAlias(`Host("localhost")`, `Host("vulcand.net")`)
	r.AddAlias(`Host("vulcand.net")`, `Host("mailgun.net")`)

	err := r.InitHandlers(map[string]interface{}{`Host("localhost") && Path("/p")`: http.HandlerFunc(f)})
	s.Require().NoError(err)

	// Should continue to route non aliased requests
	w := newWriter()
	r.ServeHTTP(w, makeReq(req{url: "/p", host: "localhost"}))
	s.Equal(http.StatusCreated, w.header)

	// Should route requests to /g for vulcand.net
	r.ServeHTTP(w, makeReq(req{url: "/p", host: "mailgun.net"}))
	s.Equal(http.StatusCreated, w.header)
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

func (w *testWriter) Header() http.Header {
	return w.headers
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	return w.buf.Write(p)
}

func (w *testWriter) WriteString(s string) (n int, err error) {
	return w.buf.WriteString(s)
}

func (w *testWriter) WriteHeader(h int) {
	w.header = h
}
