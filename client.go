// Package kuchniavikinga is an unofficial Go client for the customer API of
// panel.kuchniavikinga.pl (Kuchnia Vikinga diet catering).
//
// The API is private and undocumented by the vendor; this package replays what
// the web panel does in a browser, so it may break whenever the panel changes.
//
// Typical flow:
//
//	c := kuchniavikinga.New(kuchniavikinga.Options{})
//	if err := c.Open(ctx, kuchniavikinga.Auth{Username: u, Password: p}); err != nil { ... }
//	defer c.Close(ctx)
//	orders, _ := c.ActiveOrders(ctx)
//	order, _ := c.Order(ctx, orders[0])
//	menu, _ := c.Menu(ctx, order.Deliveries[0].ID())
//
// A Client is safe for concurrent use after Open has returned.
package kuchniavikinga

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production panel address.
const DefaultBaseURL = "https://panel.kuchniavikinga.pl"

// Auth holds the panel account credentials (the same e-mail/login and
// password used on the website).
type Auth struct {
	Username string
	Password string
}

func (a Auth) encode() string {
	form := url.Values{}
	form.Set("username", a.Username)
	form.Set("password", a.Password)
	return form.Encode()
}

// Options configures a Client. The zero value is valid.
type Options struct {
	// BaseURL overrides DefaultBaseURL (e.g. an httptest server in tests).
	BaseURL string
	// Timeout is the per-request timeout; defaults to 30s. Ignored when
	// HTTPClient is set.
	Timeout time.Duration
	// HTTPClient, if set, is used for all requests. Its Jar is replaced with
	// a fresh cookie jar if nil, because the session lives in cookies.
	// Use it to plug in a custom Transport (caching, proxies, metrics).
	HTTPClient *http.Client
	// DebugOutput, if non-nil, receives a dump of every request and response.
	// WARNING: the login request contains the password in clear text.
	DebugOutput io.Writer
}

// StatusError is returned when the panel answers with an unexpected HTTP status.
type StatusError struct {
	Method string
	URL    string
	Got    int    // status received
	Want   int    // status expected
	Body   string // response body, for diagnostics
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("%s %s: got %d %s, want %d; body: %s",
		e.Method, e.URL, e.Got, http.StatusText(e.Got), e.Want, e.Body)
}

// Client talks to the panel. Create it with New, then call Open.
type Client struct {
	http    *http.Client
	headers http.Header
	baseURL string
}

// New creates a Client. No network traffic happens until Open.
func New(opts Options) *Client {
	base := strings.TrimRight(opts.BaseURL, "/")
	if base == "" {
		base = DefaultBaseURL
	}
	hc := opts.HTTPClient
	if hc == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		hc = &http.Client{Timeout: timeout}
	} else {
		cp := *hc
		hc = &cp
	}
	if hc.Jar == nil {
		hc.Jar, _ = cookiejar.New(nil)
	}
	if opts.DebugOutput != nil {
		next := hc.Transport
		if next == nil {
			next = http.DefaultTransport
		}
		hc.Transport = &dumpTransport{next: next, out: opts.DebugOutput}
	}
	host := strings.TrimPrefix(strings.TrimPrefix(base, "https://"), "http://")
	return &Client{
		http:    hc,
		baseURL: base,
		headers: http.Header{
			"User-Agent":      {"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:137.0) Gecko/20100101 Firefox/137.0"},
			"Accept":          {"*/*"},
			"Accept-Language": {"pl,en-US;q=0.7,en;q=0.3"},
			"Referer":         {base + "/logowanie"},
			"Origin":          {base},
			"company-id":      {"kuchniavikinga"},
			"x-launcher-type": {"BROWSER_PANEL"},
			"traceparent":     {"aaaa1"},
			"Sec-Fetch-Dest":  {"empty"},
			"Sec-Fetch-Mode":  {"cors"},
			"Sec-Fetch-Site":  {"same-origin"},
			"Connection":      {"keep-alive"},
			"Alt-Used":        {host},
			"DNT":             {"1"},
			"Sec-GPC":         {"1"},
			"Priority":        {"u=0"},
		},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	maps.Copy(req.Header, c.headers)
	// The panel uses Spring CSRF protection: state-changing requests must echo
	// the XSRF-TOKEN cookie in the X-XSRF-TOKEN header, or they get 403. The
	// token is rotated after login, so it is read on every request.
	if method != http.MethodGet {
		for _, ck := range c.http.Jar.Cookies(req.URL) {
			if ck.Name == "XSRF-TOKEN" {
				req.Header.Set("X-XSRF-TOKEN", ck.Value)
			}
		}
	}
	return req, nil
}

// do sends the request and fails with *StatusError unless the response status
// equals want. The caller must close the body of a successful response.
func (c *Client) do(req *http.Request, want int) (*http.Response, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, &StatusError{Method: req.Method, URL: req.URL.String(), Got: resp.StatusCode, Want: want, Body: string(body)}
	}
	return resp, nil
}

// Open logs in by replaying the browser login sequence
// (GET /logowanie, GET /api/profile/accounts expecting 401,
// GET /api/open/analytical-tools-settings, POST /api/auth/login) and keeps the
// session cookies inside the Client. Call Close when done.
func (c *Client) Open(ctx context.Context, auth Auth) error {
	steps := []struct {
		method, path string
		body         string
		want         int
	}{
		{"GET", "/logowanie", "", http.StatusOK},
		{"GET", "/api/profile/accounts", "", http.StatusUnauthorized},
		{"GET", "/api/open/analytical-tools-settings?companyName=kuchniavikinga&host=" + url.QueryEscape(strings.TrimPrefix(strings.TrimPrefix(c.baseURL, "https://"), "http://")), "", http.StatusOK},
		{"POST", "/api/auth/login", auth.encode(), http.StatusOK},
	}
	for _, s := range steps {
		req, err := c.newRequest(ctx, s.method, s.path, strings.NewReader(s.body))
		if err != nil {
			return err
		}
		if s.body != "" {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
		resp, err := c.do(req, s.want)
		if err != nil {
			return fmt.Errorf("login: %w", err)
		}
		resp.Body.Close()
	}
	return nil
}

// Close ends the session (POST /api/auth/logout).
func (c *Client) Close(ctx context.Context) error {
	req, err := c.newRequest(ctx, "POST", "/api/auth/logout", nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req, http.StatusOK)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func requestJSON[T any](ctx context.Context, c *Client, method, path string) (*T, error) {
	req, err := c.newRequest(ctx, method, path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.do(req, http.StatusOK)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("%s %s: decode: %w", method, path, err)
	}
	return &out, nil
}

type dumpTransport struct {
	next http.RoundTripper
	out  io.Writer
}

func (t *dumpTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		fmt.Fprintf(t.out, "=== HTTP REQUEST ===\n%s\n", dump)
	}
	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		fmt.Fprintf(t.out, "=== HTTP RESPONSE ===\n%s\n", dump)
	}
	return resp, nil
}
