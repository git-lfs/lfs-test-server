package main

import (
  "net/http"
  "regexp"
  "strings"
)

type handler func(http.ResponseWriter, *http.Request, map[string]string)

type Router struct {
  routes []*Route
}

func NewRouter() *Router {
  return &Router{}
}

func (rtr *Router) ServeHTTP(w http.ResponseWriter, r *http.Request)  {
  for _, route := range rtr.routes {
    if route.handle(w, r) {
      return
    }
  }

  w.WriteHeader(404)
}

func (r *Router) Route(pattern string) *Route {
  route := newRoute(pattern)
  r.routes = append(r.routes, route)
  return route
}

type Route struct {
  pattern string
  matcher *regexp.Regexp
  variables map[string]int
  handlers map[string]handler
}

func newRoute(pattern string) *Route {
  variables := make(map[string]int)
  parts := strings.Split(pattern, "/")
  matchers := make([]string, 0, len(parts))

  for pos, part := range parts {
    if len(part) > 0 && part[0] == '{' && part[len(part)-1] == '}' {
      name := part[1:len(part)-1]
      variables[name] = pos
      matchers = append(matchers, "[^\\/]+")
    } else {
      matchers = append(matchers, part)
    }
  }

  matcherString := strings.Join(matchers, "\\/")
  r := regexp.MustCompile(matcherString)

  return &Route{pattern, r, variables, make(map[string]handler)}
}


func (rt *Route) handle(w http.ResponseWriter, r *http.Request) bool {
  mt := r.Header.Get("Accept") // remove anything after ;
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

  h(w, r, vars)

  return true
}

func (rt *Route) Get(mediaType string, h handler)  {
  rt.handlers[mediaType+"GET"] = h
}

func (rt *Route) Head(mediaType string, h handler)  {
  rt.handlers[mediaType+"HEAD"] = h

}

func (rt *Route) Post(mediaType string, h handler)  {
  rt.handlers[mediaType+"POST"] = h
}

func (rt *Route) Put(mediaType string, h handler)  {
  rt.handlers[mediaType+"PUT"] = h
}

func (rt *Route) Options(mediaType string, h handler)  {
  rt.handlers[mediaType+"OPTIONS"] = h
}
