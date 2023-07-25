# 💫 Orbit 🪐

#### Version: 0.0.1

### HTTP wrapper for golangs net/http package

#### Who is this for?

This library is for my personal usage and knowledge

#### Features [not all baked in yet]

- Context (Bits) wrapper
  - custom state injection
  - custom templating
- All handlers return an error
- HTTP helper functions
- Lite wrapper over net/http

#### Is this production ready?

No :D - this may never be production ready; a good router is a long time away

### Installation

```sh
go get github.com/ArminasAer/orbit
```

### Example

```go
package main

import (
	"net/http"

	"github.com/ArminasAer/orbit"
)

func main() {
	o := orbit.NewPlanet()

	o.Use(http.MethodGet, "/", index)

	o.Launch("127.0.0.1:9000")
}

func index(b orbit.Bits) error {
	return b.Text(http.StatusOK, "this is an orbit response")
}
```

#### Inspired by:

- Echo
  - all handlers return an error
  - context interface paradigm
- Fiber
  - simple design
  - helper functions
- Express
  - express like patterns
- Axum
  - axum global state management
