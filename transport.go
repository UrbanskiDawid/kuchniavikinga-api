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
	"strings"
)

// transport is the low-level HTTP layer: browser-like headers, cookie session,
// CSRF echo, status checking and JSON decoding.
type transport struct {
	http    *http.Client
	headers http.Header
	base    string
}

func newTransport(cfg config) *transport {
	base := strings.TrimRight(cfg.baseURL, "/")
	hc := cfg.client
	if hc == nil {
		hc = &http.Client{Timeout: cfg.timeout}
	} else {
		cp := *hc
		hc = &cp
	}
	if hc.Jar == nil {
		hc.Jar, _ = cookiejar.New(nil)
	}
	if cfg.debug != nil {
		next := hc.Transport
		if next == nil {
			next = http.DefaultTransport
		}
		hc.Transport = &dumper{next: next, out: cfg.debug}
	}
	t := &transport{http: hc, base: base}
	t.headers = http.Header{
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
		"Alt-Used":        {t.host()},
		"DNT":             {"1"},
		"Sec-GPC":         {"1"},
		"Priority":        {"u=0"},
	}
	return t
}

func (t *transport) host() string {
	return strings.TrimPrefix(strings.TrimPrefix(t.base, "https://"), "http://")
}

func (t *transport) request(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, t.base+path, body)
	if err != nil {
		return nil, err
	}
	maps.Copy(req.Header, t.headers)
	if body != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	// Spring CSRF: state-changing requests must echo the XSRF-TOKEN cookie in
	// the X-XSRF-TOKEN header. The token rotates on login, so read it each time.
	if method != http.MethodGet {
		for _, ck := range t.http.Jar.Cookies(req.URL) {
			if ck.Name == "XSRF-TOKEN" {
				req.Header.Set("X-XSRF-TOKEN", ck.Value)
			}
		}
	}
	return req, nil
}

// do sends req and returns the response, or a *StatusError if the status is not want.
func (t *transport) do(req *http.Request, want int) (*http.Response, error) {
	resp, err := t.http.Do(req)
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

// send performs a request and discards the body.
func (t *transport) send(ctx context.Context, method, path string, body *strings.Reader, want int) error {
	var r io.Reader
	if body != nil {
		r = body
	}
	req, err := t.request(ctx, method, path, r)
	if err != nil {
		return err
	}
	resp, err := t.do(req, want)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// get performs a request expecting 200 and decodes the JSON body into a T.
func get[T any](ctx context.Context, t *transport, method, path string) (v T, err error) {
	req, err := t.request(ctx, method, path, nil)
	if err != nil {
		return v, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := t.do(req, http.StatusOK)
	if err != nil {
		return v, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("%s %s: decode: %w", method, path, err)
	}
	return v, nil
}

type dumper struct {
	next http.RoundTripper
	out  io.Writer
}

func (d *dumper) RoundTrip(req *http.Request) (*http.Response, error) {
	if dump, err := httputil.DumpRequestOut(req, true); err == nil {
		fmt.Fprintf(d.out, "=== HTTP REQUEST ===\n%s\n", dump)
	}
	resp, err := d.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if dump, err := httputil.DumpResponse(resp, true); err == nil {
		fmt.Fprintf(d.out, "=== HTTP RESPONSE ===\n%s\n", dump)
	}
	return resp, nil
}
