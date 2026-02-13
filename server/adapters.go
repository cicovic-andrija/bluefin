package server

import "net/http"

// TODO: r/o is not enforced, implement a proxy that exposes a r/o interface
func readonlyHandler(fn func(http.ResponseWriter, *http.Request, *DiveLog)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, acquireDataAccess())
	}
}

// Functions below are useful, but currently not used.

// Adapter is an HTTP(S) handler that invokes another HTTP(S) handler.
type Adapter func(h http.Handler) http.Handler

// Adapt returns an HTTP(S) handler enhanced by a number of adapters.
func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
	for _, adapter := range adapters {
		h = adapter(h)
	}
	return h
}

// StripPrefix returns an adapter that calls http.StripPrefix
// to remove the given prefix from the request's URL path and invoke
// the handler h.
func StripPrefix(prefix string) Adapter {
	return func(h http.Handler) http.Handler {
		return http.StripPrefix(prefix, h)
	}
}
