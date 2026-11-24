package kuchniavikinga

import (
	"io"
	"net/http"
	"time"
)

// Option customises Login.
type Option func(*config)

type config struct {
	baseURL string
	timeout time.Duration
	client  *http.Client
	debug   io.Writer
}

func newConfig(opts []Option) config {
	cfg := config{baseURL: DefaultBaseURL, timeout: 30 * time.Second}
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

// DefaultBaseURL is the production panel address.
const DefaultBaseURL = "https://panel.kuchniavikinga.pl"

// WithBaseURL points the client at another server, e.g. an httptest server.
func WithBaseURL(u string) Option { return func(c *config) { c.baseURL = u } }

// WithTimeout sets the per-request timeout (default 30s). It is ignored when
// WithHTTPClient is used.
func WithTimeout(d time.Duration) Option { return func(c *config) { c.timeout = d } }

// WithHTTPClient supplies the http.Client, e.g. to add a caching or proxying
// Transport. The client is copied; a cookie jar is added if it has none,
// because the session lives in cookies.
func WithHTTPClient(h *http.Client) Option { return func(c *config) { c.client = h } }

// WithDebug dumps every request and response to w.
// WARNING: the login request contains the password in clear text.
func WithDebug(w io.Writer) Option { return func(c *config) { c.debug = w } }
