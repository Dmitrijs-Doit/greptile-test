package web

import (
	"net/http"
	"strings"
)

// Group wraps the App for wrapping multiple handlers with middlewares.
type Group struct {
	app         *App
	prefixPath  string
	middlewares []Middleware
}

// NewGroup initializes a group of http handlers, with a bunch of middlewares.
func NewGroup(app *App, prefixPath string, mw ...Middleware) *Group {
	return &Group{
		app,
		prefixPath,
		mw,
	}
}

// Handle uses our app.Handle mechanism for mounting Handlers for a given HTTP verb and path pair.
// it wraps a group of handlers with the given middlewares.
func (g *Group) Handle(verb string, path string, handler Handler, mw ...Middleware) {
	middlewares := append(g.middlewares, mw...)
	g.app.Handle(verb, g.prefixPath+path, handler, middlewares...)
}

// Post executes a http POST request, within a group, with the given handlers.
func (g *Group) Post(path string, handler Handler, mw ...Middleware) {
	middlewares := append(g.middlewares, mw...)
	g.app.Post(g.prefixPath+path, handler, middlewares...)
}

// Get executes a http GET request, within a group, with the given handlers.
func (g *Group) Get(path string, handler Handler, mw ...Middleware) {
	middlewares := append(g.middlewares, mw...)
	g.app.Get(g.prefixPath+path, handler, middlewares...)
}

// Put executes a http PUT request, within a group, with the given handlers.
func (g *Group) Put(path string, handler Handler, mw ...Middleware) {
	middlewares := append(g.middlewares, mw...)
	g.app.Put(g.prefixPath+path, handler, middlewares...)
}

// Delete executes a http DELETE request, within a group, with the given handlers.
func (g *Group) Delete(path string, handler Handler, mw ...Middleware) {
	middlewares := append(g.middlewares, mw...)
	g.app.Delete(g.prefixPath+path, handler, middlewares...)
}

// Patch executes a http PATCH request, within a group, with the given handlers.
func (g *Group) Patch(path string, handler Handler, mw ...Middleware) {
	middlewares := append(g.middlewares, mw...)
	g.app.Patch(g.prefixPath+path, handler, middlewares...)
}

// NewSubgroup initializes a subgroup, within a group, with a bunch of additional middlewares.
func (g *Group) NewSubgroup(prefixPath string, mw ...Middleware) *Group {
	// concatenating the group prefix path with the subgroup prefix path.
	path := strings.Join([]string{
		g.prefixPath,
		prefixPath,
	}, "")

	// appending the group middlewares with the subgroup middlewares.
	middlewares := append(g.middlewares, mw...)

	return &Group{
		g.app,
		path,
		middlewares,
	}
}

func (g *Group) Proxy(path string, handler Handler, mw ...Middleware) {
	middlewares := append(g.middlewares, mw...)
	g.app.Handle(http.MethodGet, g.prefixPath+path, handler, middlewares...)
	g.app.Handle(http.MethodPost, g.prefixPath+path, handler, middlewares...)
	g.app.Handle(http.MethodPut, g.prefixPath+path, handler, middlewares...)
	g.app.Handle(http.MethodPatch, g.prefixPath+path, handler, middlewares...)
	g.app.Handle(http.MethodDelete, g.prefixPath+path, handler, middlewares...)
	g.app.Handle(http.MethodHead, g.prefixPath+path, handler, middlewares...)
	g.app.Handle(http.MethodOptions, g.prefixPath+path, handler, middlewares...)
}
