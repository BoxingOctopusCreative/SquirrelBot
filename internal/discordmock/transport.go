package discordmock

import (
	"net/http"
	"net/url"
)

// NewTransport sends Discord HTTP calls to dest.
// Requests for any other host keep their original URL.
func NewTransport(base http.RoundTripper, dest *url.URL) http.RoundTripper {
	return transport{base: base, dest: dest}
}

type transport struct {
	base http.RoundTripper
	dest *url.URL
}

func (rt transport) RoundTrip(req *http.Request) (*http.Response, error) {
	next := req
	if req.URL.Hostname() == "discord.com" {
		next = req.Clone(req.Context())
		next.URL.Scheme = rt.dest.Scheme
		next.URL.Host = rt.dest.Host
		next.Host = rt.dest.Host
	}
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(next)
}
