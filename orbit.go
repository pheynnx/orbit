package orbit

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

var _ Router = &Orbit{}

type Handler interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type HandlerFunc func(b Bits) error

func (f HandlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := &bits{response: w, request: r}
	f(b)
}

// Top level application struct
type Orbit struct {
	handler                 Handler
	tree                    *node
	methodNotAllowedHandler HandlerFunc
	parent                  *Orbit
	pool                    *sync.Pool
	notFoundHandler         HandlerFunc
	middlewares             []func(Handler) Handler
	inline                  bool
}

// NewOrbit returns a newly initialized Orbit object that implements the Router interface
func NewOrbit() *Orbit {
	o := &Orbit{tree: &node{}, pool: &sync.Pool{}}
	o.pool.New = func() interface{} {
		return NewRouteContext()
	}
	return o
}

// Starts the server with http.ListenAndServe
func (o *Orbit) Launch(address string) error {
	fmt.Printf("💫 Orbit launching: %s 🪐\n", address)
	return http.ListenAndServe(address, o)
}

// ServeHTTP is the single method of the http.Handler interface that makes
// Orbit interoperable with the standard library. It uses a sync.Pool to get and
// reuse routing contexts for each request.
func (o *Orbit) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if o.handler == nil {
		o.NotFoundHandler().ServeHTTP(w, r)
		return
	}

	rctx, _ := r.Context().Value(RouteCtxKey).(*Context)
	if rctx != nil {
		o.handler.ServeHTTP(w, r)
		return
	}

	rctx = o.pool.Get().(*Context)
	rctx.Reset()
	rctx.Routes = o
	rctx.parentCtx = r.Context()

	r = r.WithContext(context.WithValue(r.Context(), RouteCtxKey, rctx))

	o.handler.ServeHTTP(w, r)
	o.pool.Put(rctx)
}

// Use appends a middleware handler to the Orbit middleware stack.
func (o *Orbit) Use(middlewares ...func(Handler) Handler) {
	if o.handler != nil {
		panic("orbit: all middlewares must be defined before routes on a mux")
	}
	o.middlewares = append(o.middlewares, middlewares...)
}

// Handle adds the route `pattern` that matches any http method to
// execute the `handler` http.Handler.
func (o *Orbit) Handle(pattern string, handler Handler) {
	o.handle(mALL, pattern, handler)
}

// HandleFunc adds the route `pattern` that matches any http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) HandleFunc(pattern string, handlerFn HandlerFunc) {
	o.handle(mALL, pattern, handlerFn)
}

// Method adds the route `pattern` that matches `method` http method to
// execute the `handler` http.Handler.
func (o *Orbit) Method(method, pattern string, handler Handler) {
	m, ok := methodMap[strings.ToUpper(method)]
	if !ok {
		panic(fmt.Sprintf("orbit: '%s' http method is not supported.", method))
	}
	o.handle(m, pattern, handler)
}

// MethodFunc adds the route `pattern` that matches `method` http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) MethodFunc(method, pattern string, handlerFn HandlerFunc) {
	o.Method(method, pattern, handlerFn)
}

// Connect adds the route `pattern` that matches a CONNECT http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) Connect(pattern string, handlerFn HandlerFunc) {
	o.handle(mCONNECT, pattern, handlerFn)
}

// Delete adds the route `pattern` that matches a DELETE http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) Delete(pattern string, handlerFn HandlerFunc) {
	o.handle(mDELETE, pattern, handlerFn)
}

// Get adds the route `pattern` that matches a GET http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) Get(pattern string, handlerFn HandlerFunc) {
	o.handle(mGET, pattern, handlerFn)
}

// Head adds the route `pattern` that matches a HEAD http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) Head(pattern string, handlerFn HandlerFunc) {
	o.handle(mHEAD, pattern, handlerFn)
}

// Options adds the route `pattern` that matches an OPTIONS http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) Options(pattern string, handlerFn HandlerFunc) {
	o.handle(mOPTIONS, pattern, handlerFn)
}

// Patch adds the route `pattern` that matches a PATCH http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) Patch(pattern string, handlerFn HandlerFunc) {
	o.handle(mPATCH, pattern, handlerFn)
}

// Post adds the route `pattern` that matches a POST http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) Post(pattern string, handlerFn HandlerFunc) {
	o.handle(mPOST, pattern, handlerFn)
}

// Put adds the route `pattern` that matches a PUT http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) Put(pattern string, handlerFn HandlerFunc) {
	o.handle(mPUT, pattern, handlerFn)
}

// Trace adds the route `pattern` that matches a TRACE http method to
// execute the `handlerFn` http.HandlerFunc.
func (o *Orbit) Trace(pattern string, handlerFn HandlerFunc) {
	o.handle(mTRACE, pattern, handlerFn)
}

// TODO
// NotFound sets a custom http.HandlerFunc for routing paths that could
// not be found. The default 404 handler is `http.NotFound`.
func (o *Orbit) NotFound(handlerFn HandlerFunc) {
	m := o
	hFn := handlerFn

	if o.inline && o.parent != nil {
		m = o.parent
		// hFn = Chain(o.middlewares...).HandlerFunc(hFn).ServeHTTP
	}

	// Update the notFoundHandler from this point forward
	m.notFoundHandler = hFn
	m.updateSubRoutes(func(subMux *Orbit) {
		if subMux.notFoundHandler == nil {
			subMux.NotFound(hFn)
		}
	})
}

// TODO
// MethodNotAllowed sets a custom http.HandlerFunc for routing paths where the
// method is unresolved. The default handler returns a 405 with an empty body.
func (o *Orbit) MethodNotAllowed(handlerFn HandlerFunc) {
	// Build MethodNotAllowed handler chain
	m := o
	hFn := handlerFn
	if o.inline && o.parent != nil {
		m = o.parent
		// hFn = Chain(o.middlewares...).HandlerFunc(hFn).ServeHTTP
	}

	m.methodNotAllowedHandler = hFn
	m.updateSubRoutes(func(subMux *Orbit) {
		if subMux.methodNotAllowedHandler == nil {
			subMux.MethodNotAllowed(hFn)
		}
	})
}

// With adds inline middlewares for an endpoint handler.
func (o *Orbit) With(middlewares ...func(Handler) Handler) Router {
	if !o.inline && o.handler == nil {
		o.updateRouteHandler()
	}

	var mws Middlewares
	if o.inline {
		mws = make(Middlewares, len(o.middlewares))
		copy(mws, o.middlewares)
	}
	mws = append(mws, middlewares...)

	im := &Orbit{
		pool: o.pool, inline: true, parent: o, tree: o.tree, middlewares: mws,
		notFoundHandler: o.notFoundHandler, methodNotAllowedHandler: o.methodNotAllowedHandler,
	}

	return im
}

// Group creates a new inline-Mux with a fresh middleware stack. It's useful
// for a group of handlers along the same routing path that use an additional
// set of middlewares.
func (o *Orbit) Group(fn func(r Router)) Router {
	im := o.With().(*Orbit)
	if fn != nil {
		fn(im)
	}
	return im
}

// Route creates a new Orbit with a fresh middleware stack and mounts it
// along the `pattern` as a subrouter. Effectively, this is a short-hand
// call to Mount.
func (o *Orbit) Route(pattern string, fn func(r Router)) Router {
	if fn == nil {
		panic(fmt.Sprintf("orbit: attempting to Route() a nil subrouter on '%s'", pattern))
	}
	subRouter := NewPlanet()
	fn(subRouter)
	o.Mount(pattern, subRouter)
	return subRouter
}

// Mount attaches another http.Handler or orbit Router as a subrouter along a routing
// path. It's very useful to split up a large API as many independent routers and
// compose them as a single service using Mount. See _examples/.
//
// Note that Mount() simply sets a wildcard along the `pattern` that will continue
// routing at the `handler`, which in most cases is another orbit.Router. As a result,
// if you define two Mount() routes on the exact same pattern the mount will panic.
func (o *Orbit) Mount(pattern string, handler Handler) {
	if handler == nil {
		panic(fmt.Sprintf("orbit: attempting to Mount() a nil handler on '%s'", pattern))
	}

	if o.tree.findPattern(pattern+"*") || o.tree.findPattern(pattern+"/*") {
		panic(fmt.Sprintf("orbit: attempting to Mount() a handler on an existing path, '%s'", pattern))
	}

	subr, ok := handler.(*Orbit)
	if ok && subr.notFoundHandler == nil && o.notFoundHandler != nil {
		subr.NotFound(o.notFoundHandler)
	}
	if ok && subr.methodNotAllowedHandler == nil && o.methodNotAllowedHandler != nil {
		subr.MethodNotAllowed(o.methodNotAllowedHandler)
	}

	mountHandler := HandlerFunc(func(b Bits) error {
		rctx := RouteContext(b.Request().Context())

		rctx.RoutePath = o.nextRoutePath(rctx)
		n := len(rctx.URLParams.Keys) - 1
		if n >= 0 && rctx.URLParams.Keys[n] == "*" && len(rctx.URLParams.Values) > n {
			rctx.URLParams.Values[n] = ""
		}

		handler.ServeHTTP(b.Response(), b.Request())

		return nil
	})

	if pattern == "" || pattern[len(pattern)-1] != '/' {
		o.handle(mALL|mSTUB, pattern, mountHandler)
		o.handle(mALL|mSTUB, pattern+"/", mountHandler)
		pattern += "/"
	}

	method := mALL
	subroutes, _ := handler.(Routes)
	if subroutes != nil {
		method |= mSTUB
	}
	n := o.handle(method, pattern+"*", mountHandler)

	if subroutes != nil {
		n.subroutes = subroutes
	}
}

// Routes returns a slice of routing information from the tree,
// useful for traversing available routes of a router.
func (o *Orbit) Routes() []Route {
	return o.tree.routes()
}

// Middlewares returns a slice of middleware handler functions.
func (o *Orbit) Middlewares() Middlewares {
	return o.middlewares
}

// Match searches the routing tree for a handler that matches the method/path.
// It's similar to routing a http request, but without executing the handler
// thereaf
func (o *Orbit) Match(rctx *Context, method, path string) bool {
	m, ok := methodMap[method]
	if !ok {
		return false
	}

	node, _, h := o.tree.FindRoute(rctx, m, path)

	if node != nil && node.subroutes != nil {
		rctx.RoutePath = o.nextRoutePath(rctx)
		return node.subroutes.Match(rctx, method, rctx.RoutePath)
	}

	return h != nil
}

// NotFoundHandler returns the default Orbit 404 responder whenever a route
// cannot be found.
func (o *Orbit) NotFoundHandler() HandlerFunc {
	if o.notFoundHandler != nil {
		return o.notFoundHandler
	}
	// TODO
	return func(b Bits) error {
		http.Error(b.Response(), "404 page not found", http.StatusNotFound)
		return nil
	}
}

// MethodNotAllowedHandler returns the default Orbit 405 responder whenever
// a method cannot be resolved for a route.
func (o *Orbit) MethodNotAllowedHandler(methodsAllowed ...methodTyp) HandlerFunc {
	if o.methodNotAllowedHandler != nil {
		return o.methodNotAllowedHandler
	}
	return methodNotAllowedHandler(methodsAllowed...)
}

// handle registers a http.Handler in the routing tree for a particular http method
// and routing pattern.
func (o *Orbit) handle(method methodTyp, pattern string, handler http.Handler) *node {
	if len(pattern) == 0 || pattern[0] != '/' {
		panic(fmt.Sprintf("orbit: routing pattern must begin with '/' in '%s'", pattern))
	}

	if !o.inline && o.handler == nil {
		o.updateRouteHandler()
	}

	var h http.Handler
	if o.inline {
		o.handler = http.HandlerFunc(o.routeHTTP)
		h = Chain(o.middlewares...).Handler(handler)
	} else {
		h = handler
	}
	return o.tree.InsertRoute(method, pattern, h)
}

// routeHTTP routes a http.Request through the Orbit routing tree to serve
// the matching handler for a particular http method.
func (o *Orbit) routeHTTP(w http.ResponseWriter, r *http.Request) {
	rctx := r.Context().Value(RouteCtxKey).(*Context)

	routePath := rctx.RoutePath
	if routePath == "" {
		if r.URL.RawPath != "" {
			routePath = r.URL.RawPath
		} else {
			routePath = r.URL.Path
		}
		if routePath == "" {
			routePath = "/"
		}
	}

	if rctx.RouteMethod == "" {
		rctx.RouteMethod = r.Method
	}
	method, ok := methodMap[rctx.RouteMethod]
	if !ok {
		o.MethodNotAllowedHandler().ServeHTTP(w, r)
		return
	}

	if _, _, h := o.tree.FindRoute(rctx, method, routePath); h != nil {
		h.ServeHTTP(w, r)
		return
	}
	if rctx.methodNotAllowed {
		o.MethodNotAllowedHandler(rctx.methodsAllowed...).ServeHTTP(w, r)
	} else {
		o.NotFoundHandler().ServeHTTP(w, r)
	}
}

func (o *Orbit) nextRoutePath(rctx *Context) string {
	routePath := "/"
	nx := len(rctx.routeParams.Keys) - 1
	if nx >= 0 && rctx.routeParams.Keys[nx] == "*" && len(rctx.routeParams.Values) > nx {
		routePath = "/" + rctx.routeParams.Values[nx]
	}
	return routePath
}

// Recursively update data on child routers.
func (o *Orbit) updateSubRoutes(fn func(subMux *Orbit)) {
	for _, r := range o.tree.routes() {
		subMux, ok := r.SubRoutes.(*Orbit)
		if !ok {
			continue
		}
		fn(subMux)
	}
}

// updateRouteHandler builds the single mux handler that is a chain of the middleware
// stack, as defined by calls to Use(), and the tree router (Orbit) itself. After this
// point, no other middlewares can be registered on this Orbit's stack. But you can still
// compose additional middlewares via Group()'s or using a chained middleware handler.
func (o *Orbit) updateRouteHandler() {
	o.handler = chain(o.middlewares, http.HandlerFunc(o.routeHTTP))
}

// methodNotAllowedHandler is a helper function to respond with a 405,
// method not allowed. It sets the Allow header with the list of allowed
// methods for the route.
func methodNotAllowedHandler(methodsAllowed ...methodTyp) HandlerFunc {

	return func(b Bits) error {
		for _, m := range methodsAllowed {

			b.Response().Header().Add("Allow", reverseMethodMap[m])
		}
		b.Response().WriteHeader(405)
		_, err := b.Response().Write(nil)

		return err

	}
}
