package orbit

import "net/http"

type Bits interface {
	Response() http.ResponseWriter
	Request() *http.Request
	Text(code int, s string) error
}

type bits struct {
	response http.ResponseWriter
	request  *http.Request
	name     string
}

func (b *bits) Response() http.ResponseWriter {
	return b.response
}

func (b *bits) Request() *http.Request {
	return b.request
}

func (b *bits) Text(code int, s string) error {
	b.response.Header().Set("Content-Type", "text/plain")
	b.response.WriteHeader(code)
	_, err := b.response.Write([]byte(s))
	return err
}

// // TODO
// func (b *bits) Html(code int, html string) error {
// 	b.response.Header().Set("Content-Type", "text/html")
// 	b.response.WriteHeader(code)
// 	return nil
// }

// // TODO
// func (b *bits) Json(code int, json any) error {
// 	b.response.Header().Set("Content-Type", "application/json")
// 	b.response.WriteHeader(code)
// 	return nil
// }
