package orbit

import (
	"fmt"
	"net/http"
)

type Handler func(b Bits) error

type Route struct {
	Method  string
	Path    string
	Handler Handler
}

// Top level http wrapper
type Orbit struct {
	routes       []Route
	DefaultRoute Handler
}

// Create new instance of Orbit
func NewPlanet() *Orbit {
	// TODO
	// Allow for configuration to be set from constructor
	app := &Orbit{
		DefaultRoute: func(b Bits) error {
			return b.Text(http.StatusNotFound, "path not found")
		},
	}

	return app
}

// Serve the application
func (a *Orbit) Launch(address string) error {
	fmt.Printf("Orbit launching: %s\n", address)
	return http.ListenAndServe(address, a)
}

// TODO
func (a *Orbit) Use(method string, path string, handler Handler) {
	// re := regexp.MustCompile(pattern)
	route := Route{Path: path, Handler: handler}

	a.routes = append(a.routes, route)
}

// TODO
func (a *Orbit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := &bits{request: r, response: w}

	for _, rt := range a.routes {
		if rt.Path == r.URL.Path {
			rt.Handler(b)
			return
		}
	}

	a.DefaultRoute(b)
}
