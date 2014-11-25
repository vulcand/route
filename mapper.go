package route

import (
	"net/http"
	"strings"
)

// requestMapper maps the request to string
type requestMapper interface {
	// separator returns the separator that makes sense for this request, e.g. / for urls or . for domains
	separator() byte
	// equals returns true if two mappers are equivalent
	equals(requestMapper) bool
	// mapRequest maps request to string, e.g. request to it's URL path
	mapRequest(r *http.Request) string
	// newIter returns the iterator instead of string for stream matchers
	newIter(r *http.Request) *charIter
}

type methodMapper struct {
}

func (m *methodMapper) separator() byte {
	return methodSep
}

func (m *methodMapper) equals(o requestMapper) bool {
	_, ok := o.(*methodMapper)
	return ok
}

func (m *methodMapper) mapRequest(r *http.Request) string {
	return r.Method
}

func (m *methodMapper) newIter(r *http.Request) *charIter {
	return newIter([]string{m.mapRequest(r)}, []byte{m.separator()})
}

type pathMapper struct {
}

func (m *pathMapper) separator() byte {
	return pathSep
}

func (p *pathMapper) equals(o requestMapper) bool {
	_, ok := o.(*pathMapper)
	return ok
}

func (p *pathMapper) newIter(r *http.Request) *charIter {
	return newIter([]string{p.mapRequest(r)}, []byte{p.separator()})
}

func (p *pathMapper) mapRequest(r *http.Request) string {
	path, err := RawPath(r.RequestURI)
	if err != nil {
		path = r.URL.Path
	}
	if len(path) == 0 {
		return "/"
	}
	return path
}

type hostMapper struct {
}

func (p *hostMapper) equals(o requestMapper) bool {
	_, ok := o.(*hostMapper)
	return ok
}

func (m *hostMapper) separator() byte {
	return domainSep
}

func (h *hostMapper) mapRequest(r *http.Request) string {
	return strings.Split(strings.ToLower(r.Host), ":")[0]
}

func (p *hostMapper) newIter(r *http.Request) *charIter {
	return newIter([]string{p.mapRequest(r)}, []byte{p.separator()})
}

type headerMapper struct {
	header string
}

func (h *headerMapper) equals(o requestMapper) bool {
	hm, ok := o.(*headerMapper)
	return ok && hm.header == h.header
}

func (m *headerMapper) separator() byte {
	return headerSep
}

func (h *headerMapper) mapRequest(r *http.Request) string {
	return r.Header.Get(h.header)
}

func (h *headerMapper) newIter(r *http.Request) *charIter {
	return newIter([]string{h.mapRequest(r)}, []byte{h.separator()})
}

type seqMapper struct {
	seq []requestMapper
}

func newSeqMapper(seq ...requestMapper) *seqMapper {
	var out []requestMapper
	for _, s := range seq {
		switch m := s.(type) {
		case *seqMapper:
			out = append(out, m.seq...)
		default:
			out = append(out, s)
		}
	}
	return &seqMapper{seq: out}
}

func (s *seqMapper) newIter(r *http.Request) *charIter {
	out := make([]string, len(s.seq))
	for i := range s.seq {
		out[i] = s.seq[i].mapRequest(r)
	}
	seps := make([]byte, len(s.seq))
	for i := range s.seq {
		seps[i] = s.seq[i].separator()
	}
	return newIter(out, seps)
}

func (s *seqMapper) mapRequest(r *http.Request) string {
	out := make([]string, len(s.seq))
	for i := range s.seq {
		out[i] = s.seq[i].mapRequest(r)
	}
	return strings.Join(out, "")
}

func (s *seqMapper) separator() byte {
	return s.seq[0].separator()
}

func (s *seqMapper) equals(o requestMapper) bool {
	so, ok := o.(*seqMapper)
	if !ok {
		return false
	}
	if len(s.seq) != len(so.seq) {
		return false
	}
	for i, _ := range s.seq {
		if !s.seq[i].equals(so.seq[i]) {
			return false
		}
	}
	return true
}

const (
	pathSep   = '/'
	domainSep = '.'
	headerSep = '/'
	methodSep = ' '
)
