package main

import (
	"net/http"

	"babylogger"
)

func main() {

	// HTTP server with Babylogger middleware
	http.Handle("/", babylogger.Middleware(http.HandlerFunc(handler)))
	go func() {
		http.ListenAndServe(":1337", nil)
	}()

	// Perform some example HTTP requests, then exit
	h := "http://localhost:1337"
	c := &http.Client{}
	c.Get(h + "/")
	r, _ := http.NewRequest("POST", h+"/meow", nil)
	c.Do(r)
	r, _ = http.NewRequest("PUT", h+"/purr", nil)
	c.Do(r)
	c.Get(h + "/schnurr")
}

func handler(w http.ResponseWriter, r *http.Request) {
	switch r.RequestURI {
	case "/":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("oh hey"))
	case "/meow":
		w.WriteHeader(http.StatusTemporaryRedirect)
		w.Write([]byte("it's over there"))
	case "/purr":
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("nope, not here"))
	default:
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("ouch"))
	}
}
