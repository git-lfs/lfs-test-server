package main

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

var (
	mutex sync.RWMutex
	rvars = make(map[*http.Request]map[string]string)
)

// Router provides a simple and unforgiving router. The router does a simplistic pattern
// matching and variable substitution, and has a focus on media types provided in the request's
// Accept header.
type Router struct {
	routes []*Route
}

// NewRouter creates a new Router
func NewRouter() *Router {
	return &Router{}
}

func (rtr *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range rtr.routes {
		if route.handle(w, r) {
			return
		}
	}

	logRequest(r, 404)
	w.WriteHeader(404)
}

// Route defines the router for the given pattern. Patterns are defined as such:
//
// 	r := Route("/users/{name}/articles/{article}")
//
// The path components surrounded by "{}" will be extracted into variables with the
// names `name` and `article`, respectively. These variables can be accessed with
// the Vars() function.
//
// Once a route is defined, Get(), Head(), Post(), Put(), and Options() can be used
// to set up handlers for a given media type.
func (rtr *Router) Route(pattern string) *Route {
	route := newRoute(pattern)
	rtr.routes = append(rtr.routes, route)
	return route
}

// Route matches paths to handlers
type Route struct {
	pattern   string
	matcher   *regexp.Regexp
	variables map[string]int
	handlers  map[string]http.HandlerFunc
}

// Get sets up a handler for a GET request of the given media type.
func (rt *Route) Get(mediaType string, h http.HandlerFunc) {
	rt.handlers[mediaType+"GET"] = h
}

// Head sets up a handler for a HEAD request of the given media type.
func (rt *Route) Head(mediaType string, h http.HandlerFunc) {
	rt.handlers[mediaType+"HEAD"] = h
}

// Post sets up a handler for a POST request of the given media type.
func (rt *Route) Post(mediaType string, h http.HandlerFunc) {
	rt.handlers[mediaType+"POST"] = h
}

// Put sets up a handler for a PUT request of the given media type.
func (rt *Route) Put(mediaType string, h http.HandlerFunc) {
	rt.handlers[mediaType+"PUT"] = h
}

// Options sets up a handler for a OPTIONS request of the given media type.
func (rt *Route) Options(mediaType string, h http.HandlerFunc) {
	rt.handlers[mediaType+"OPTIONS"] = h
}

// Vars retrieves the extracted variables for the given http.Request object.
func Vars(r *http.Request) map[string]string {
	mutex.RLock()
	v := rvars[r]
	mutex.RUnlock()
	return v
}

func newRoute(pattern string) *Route {
	variables := make(map[string]int)
	parts := strings.Split(pattern, "/")
	matchers := make([]string, 0, len(parts))

	for pos, part := range parts {
		if len(part) > 0 && part[0] == '{' && part[len(part)-1] == '}' {
			name := part[1 : len(part)-1]
			variables[name] = pos
			matchers = append(matchers, "[^\\/]+")
		} else {
			matchers = append(matchers, part)
		}
	}

	matcherString := strings.Join(matchers, "\\/")
	r := regexp.MustCompile(matcherString)

	return &Route{pattern, r, variables, make(map[string]http.HandlerFunc)}
}

func (rt *Route) handle(w http.ResponseWriter, r *http.Request) bool {
	mediaParts := strings.Split(r.Header.Get("Accept"), ";")
	mt := mediaParts[0]
	h, ok := rt.handlers[mt+r.Method]
	if !ok {
		return false
	}

	if !rt.matcher.MatchString(r.URL.Path) {
		return false
	}

	parts := strings.Split(r.URL.Path, "/")
	vars := make(map[string]string)
	for name, pos := range rt.variables {
		vars[name] = parts[pos]
	}

	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err == nil {
		vars["request_id"] = fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	}

	mutex.Lock()
	rvars[r] = vars
	mutex.Unlock()

	h(w, r)

	mutex.Lock()
	delete(rvars, r)
	mutex.Unlock()

	return true
}
