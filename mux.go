package route

import (
	"fmt"
	"net/http"
	"strings"
)

// Mux implements router compatible with http.Handler
type Mux struct {
	// NotFound sets handler for routes that are not found
	notFound http.Handler
	router   Router
	aliases  []alias
}

type alias struct {
	match   string
	replace string
}

// NewMux returns new Mux router
func NewMux() *Mux {
	return &Mux{
		router:   New(),
		notFound: &notFound{},
	}
}

// AddAlias adds an alias for matchers in an expression. If the string
// in `match` matches any part of an expression added via `Mux.Handle()`
// then the match is replaced with the value of `alias`.
func (m *Mux) AddAlias(match, replace string) {
	m.aliases = append(m.aliases, alias{match: match, replace: replace})
}

func (m *Mux) applyAliases(expr string) string {
	for _, alias := range m.aliases {
		expr = strings.Replace(expr, alias.match, alias.replace, -1)
	}
	return expr
}

// This adds a map of handlers and expressions in a single call. This allows
// init to load many rules on first startup, thus reducing the time it takes to
// create the initial mux.
func (m *Mux) InitHandlers(handlers map[string]interface{}) error {
	if len(m.aliases) == 0 {
		return m.router.InitRoutes(handlers)
	}

	// Apply aliases to routes
	modified := make(map[string]interface{}, len(handlers))
	for k, v := range handlers {
		k = m.applyAliases(k)
		modified[k] = v
	}
	return m.router.InitRoutes(modified)
}

// Handle adds http handler for route expression
func (m *Mux) Handle(expr string, handler http.Handler) error {
	expr = m.applyAliases(expr)
	return m.router.UpsertRoute(expr, handler)
}

// Handle adds http handler function for route expression
func (m *Mux) HandleFunc(expr string, handler func(http.ResponseWriter, *http.Request)) error {
	expr = m.applyAliases(expr)
	return m.Handle(expr, http.HandlerFunc(handler))
}

func (m *Mux) Remove(expr string) error {
	expr = m.applyAliases(expr)
	return m.router.RemoveRoute(expr)
}

// ServeHTTP routes the request and passes it to handler
func (m *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, err := m.router.Route(r)
	if err != nil || h == nil {
		m.notFound.ServeHTTP(w, r)
		return
	}
	h.(http.Handler).ServeHTTP(w, r)
}

func (m *Mux) SetNotFound(n http.Handler) error {
	if n == nil {
		return fmt.Errorf("Not Found handler cannot be nil. Operation rejected.")
	}
	m.notFound = n
	return nil
}

func (m *Mux) GetNotFound() http.Handler {
	return m.notFound
}

func (m *Mux) IsValid(expr string) bool {
	return IsValid(expr)
}

// NotFound is a generic http.Handler for request
type notFound struct {
}

// ServeHTTP returns a simple 404 Not found response
func (notFound) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "Not found")

}
