Babylogger
==========

A Go HTTP logger middleware, for babies.

![Example image of Babylogger doing its logging]
(https://i.imgur.com/VGg7Wl6.png)

We’ve used it with [Goji][goji] and the Go standard library, but it should work
with any multiplexer worth its salt. And by that we mean any multiplexer
compatible with the standard library.

Note that for accurate logging, babylogger should be the first middleware
called.

## Examples

### Standard Library

```go
package main

import (
    "fmt"
    "net/http"
    "github.com/meowgorithm/babylogger"
)

func main() {
    http.Handle("/", babylogger.Middleware(http.HandlerFunc(handler)))
    http.ListenAndServe(":8000", nil)
}

handler(w http.ResponseWriter, r *http.Request) {
    fmt.FPrintln(w, "Oh, hi, I didn’t see you there.")
}
```

### Goji

```go

import (
    "fmt"
    "net/http"
    "github.com/meowgorithm/babylogger"
    "goji.io"
    "goji.io/pat"
)

func main() {
    mux := goji.NewMux()
    mux.Use(babylogger.Middleware)
    mux.HandleFunc(pat.Get("/"), handler)
    http.ListenAndServe(":8000", mux)
}

handler(w http.ResponseWriter, r *http.Request) {
    fmt.FPrintln(w, "Oh, hi, I didn’t see you there.")
}
```


## License

MIT

[goji]: http://goji.io
