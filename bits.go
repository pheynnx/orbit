package orbit

import (
	"net/http"
)

type Bits interface {
	Request() *http.Request
	Response() http.ResponseWriter
	// ParamValues() []string
	Html(code int, html string) error
	Json(code int, json any) error
	Text(code int, s string) error
}

type bits struct {
	request  *http.Request
	response http.ResponseWriter
}

func (b *bits) Request() *http.Request {
	return b.request
}

func (b *bits) Response() http.ResponseWriter {
	return b.response
}

func (b *bits) Text(code int, s string) (err error) {
	b.response.Header().Set("Content-Type", "text/plain")
	b.response.WriteHeader(code)
	_, err = b.response.Write([]byte(s))
	return
}

// TODO
func (b *bits) Html(code int, html string) error {
	b.response.Header().Set("Content-Type", "text/plain")
	return nil
}

// TODO
func (b *bits) Json(code int, json any) error {
	return nil
}
