package ai

import (
	"net/http"
	"net/url"
)

// redirectTransport redirects all HTTP requests to a given test server URL.
// This allows testing providers that use hardcoded API URLs without real network calls.
type redirectTransport struct {
	serverURL string
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base, _ := url.Parse(t.serverURL)
	req = req.Clone(req.Context())
	req.URL.Scheme = base.Scheme
	req.URL.Host = base.Host
	return http.DefaultTransport.RoundTrip(req)
}

// newTestClient returns an http.Client that redirects all requests to serverURL.
func newTestClient(serverURL string) *http.Client {
	return &http.Client{
		Transport: &redirectTransport{serverURL: serverURL},
	}
}
