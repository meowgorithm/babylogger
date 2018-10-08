// Package babylogger is a simple HTTP logging middleware. It works with any
// multiplexer compatible with the Go standard library.
//
// When a terminal is present it will log using nice colors. When the output is
// not in a terminal (for example in logs) ANSI escape sequences (read: colors)
// will be stripped from the output.
//
// Also note that for accurate response time logging Babylogger should be the
// first middleware called.
//
// Windows support is not currently implemented, however it would be trivial
// enough with the help of a couple packages from Mattn:
// http://github.com/mattn/go-isatty and https://github.com/mattn/go-colorable
//
// Example using the standard library:
//
//	package main
//
//	import (
//		"fmt"
//		"net/http"
//		"github.com/magicnumbers/babylogger"
//	)
//
//	func main() {
//		http.Handle("/", babylogger.Middleware(http.HandlerFunc(handler)))
//		http.ListenAndServe(":8000", nil)
//	}
//
//	handler(w http.ResponseWriter, r *http.Request) {
//		fmt.FPrintln(w, "Oh, hi, I didn’t see you there.")
//	}
//
// Example with Goji:
//
//	import (
//		"fmt"
//		"net/http"
//		"github.com/magicnumbers/babylogger"
//		"goji.io"
//		"goji.io/pat"
//	)
//
//	func main() {
//		mux := goji.NewMux()
//		mux.Use(babylogger.Middleware)
//		mux.HandleFunc(pat.Get("/"), handler)
//		http.ListenAndServe(":8000", mux)
//	}
//
//	handler(w http.ResponseWriter, r *http.Request) {
//		fmt.FPrintln(w, "Oh hi, I didn’t see you there.")
//	}
package babylogger

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	reset = "\x1b[0m"
	fg    = "\x1b[38;5;" // bg prefix is "\x1b[48;5;"
	end   = "m"
)

// Color keys
const (
	violet = iota
	red
	yellow
	green
	cyan
	darkGrey
	grey
)

var (
	colors = map[int]string{
		violet:   fg + "62" + end,
		red:      fg + "204" + end,
		yellow:   fg + "192" + end,
		green:    fg + "48" + end,
		cyan:     fg + "86" + end,
		darkGrey: fg + "240" + end,
		grey:     fg + "250" + end,
	}
)

// See if we're in a terminal and, if so, unset color variables
func checkTTY() {
	// So we could check for a Windows terminal with:
	//     http://github.com/mattn/go-isatty
	//
	// And then, handle colors in Windows terminals with:
	//     https://github.com/mattn/go-colorable
	//
	// But we're avoiding the (popular) `go-isatty` package for now because it
	// imports the "unsafe" pacakge. It would probably be fine though.
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		for k := range colors {
			colors[k] = ""
		}
	}
}

type logWriter struct {
	http.ResponseWriter
	code, bytes int
}

func (r *logWriter) Write(p []byte) (int, error) {
	written, err := r.ResponseWriter.Write(p)
	r.bytes += written
	return written, err
}

// Note this is generally only called when sending an HTTP error, so it's
// important to set the `code` value to 200 as a default
func (r *logWriter) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

// Middleware is the logging middleware where we log incoming and outgoing
// requests for a multiplexer. It should be the first middleware called so it
// can log request times accurately.
func Middleware(next http.Handler) http.Handler {
	checkTTY()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		addr := r.RemoteAddr
		if colon := strings.LastIndex(addr, ":"); colon != -1 {
			addr = addr[:colon]
		}

		arrow := colors[darkGrey] + "<-"
		method := colors[violet] + r.Method
		uri := colors[grey] + r.RequestURI
		address := colors[darkGrey] + addr

		// Log request
		log.Printf("%s %s %s %s%s", arrow, method, uri, address, reset)

		writer := &logWriter{
			ResponseWriter: w,
			code:           http.StatusOK, // default. so important! see above.
		}

		arrow = colors[darkGrey] + "->"
		startTime := time.Now()
		next.ServeHTTP(writer, r)
		elapsedTime := time.Now().Sub(startTime)

		status := fmt.Sprintf("%d %s", writer.code, http.StatusText(writer.code))
		if writer.code < 300 { // 200s
			status = colors[green] + status
		} else if writer.code < 400 { // 300s
			status = colors[yellow] + status
		} else if writer.code < 500 { // 400s
			status = colors[cyan] + status
		} else { // 500s
			status = colors[red] + status
		}

		// The excellent humanize package adds a space between the integer and
		// the unit as far as bytes are conerned (105 B). In our case that
		// makes it a little harder on the eyes when scanning the logs, so
		// we're stripping that space
		formattedBytes := strings.Replace(
			humanize.Bytes(uint64(writer.bytes)),
			" ", "", 1)

		bytes := fmt.Sprintf("%s%s", colors[grey], formattedBytes)
		time := fmt.Sprintf("%s%v", colors[darkGrey], elapsedTime)

		// Log response
		log.Printf("%s %s %s %v%s", arrow, status, bytes, time, reset)
	})
}
