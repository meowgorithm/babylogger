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
//		"github.com/meowgorithm/babylogger"
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
//		"github.com/meowgorithm/babylogger"
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
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	humanize "github.com/dustin/go-humanize"
)

// Styles.
var (
	timeStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "240",
		Dark:  "240",
	})

	uriStyle = timeStyle.Copy()

	methodStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "62",
		Dark:  "62",
	})

	http200Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "35",
		Dark:  "48",
	})

	http300Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "208",
		Dark:  "192",
	})

	http400Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "39",
		Dark:  "86",
	})

	http500Style = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "203",
		Dark:  "204",
	})

	subtleStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: "250",
		Dark:  "250",
	})

	addressStyle = subtleStyle.Copy()
)

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

// Hijack exposes the underlying ResponseWriter Hijacker implementation for
// WebSocket compatibility
func (r *logWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("WebServer does not support hijacking")
	}
	return hj.Hijack()
}

// Middleware is the logging middleware where we log incoming and outgoing
// requests for a multiplexer. It should be the first middleware called so it
// can log request times accurately.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		addr := r.RemoteAddr
		if colon := strings.LastIndex(addr, ":"); colon != -1 {
			addr = addr[:colon]
		}

		arrow := subtleStyle.Render("<-")
		method := methodStyle.Render(r.Method)
		uri := uriStyle.Render(r.RequestURI)
		address := addressStyle.Render(addr)

		// Log request
		log.Printf("%s %s %s %s", arrow, method, uri, address)

		writer := &logWriter{
			ResponseWriter: w,
			code:           http.StatusOK, // default. so important! see above.
		}

		arrow = subtleStyle.Render("->")
		startTime := time.Now()

		// Not sure why the request could possibly be nil, but it has happened
		if r == nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError),
				http.StatusInternalServerError)
			writer.code = http.StatusInternalServerError
		} else {
			next.ServeHTTP(writer, r)
		}

		elapsedTime := time.Now().Sub(startTime)

		var statusStyle lipgloss.Style

		if writer.code < 300 { // 200s
			statusStyle = http200Style
		} else if writer.code < 400 { // 300s
			statusStyle = http300Style
		} else if writer.code < 500 { // 400s
			statusStyle = http400Style
		} else { // 500s
			statusStyle = http500Style
		}

		status := statusStyle.Render(fmt.Sprintf("%d %s", writer.code, http.StatusText(writer.code)))

		// The excellent humanize package adds a space between the integer and
		// the unit as far as bytes are conerned (105 B). In our case that
		// makes it a little harder on the eyes when scanning the logs, so
		// we're stripping that space
		formattedBytes := strings.Replace(
			humanize.Bytes(uint64(writer.bytes)),
			" ", "", 1)

		bytes := subtleStyle.Render(formattedBytes)
		time := timeStyle.Render(fmt.Sprintf("%s", elapsedTime))

		// Log response
		log.Printf("%s %s %s %v", arrow, status, bytes, time)
	})
}
