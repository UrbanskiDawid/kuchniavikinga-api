// Package kuchniavikinga is an unofficial Go client for the customer panel of
// Kuchnia Vikinga (panel.kuchniavikinga.pl), a diet catering service.
//
// The model mirrors how the panel works: you log in, and walk down from
// orders to deliveries to meals. Every level is a handle that knows how to
// fetch or change what is below it, so you never pass around raw ids.
//
//	c, err := kuchniavikinga.Login(ctx, user, pass)
//	if err != nil { ... }
//	defer c.Logout(ctx)
//
//	orders, _ := c.Orders(ctx)                   // active orders
//	for _, d := range orders[0].Deliveries {     // one per delivery day
//	    meals, _ := d.Meals(ctx)                 // breakfast, lunch, ...
//	    for _, m := range meals {
//	        alts, _ := m.Alternatives(ctx)       // what it can be swapped for
//	        _ = m.SwitchTo(ctx, alts[0])         // changes your real order
//	    }
//	}
//
// The API is private and the vendor may change it at any time; this package
// replays what the web panel does in a browser. A Client is safe for
// concurrent use.
package kuchniavikinga

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Client is a logged-in session. Create one with Login.
type Client struct {
	t *transport
}

// Login signs in and returns a ready-to-use Client. Call Logout when done.
//
// It replays the browser login sequence, which also sets the session and CSRF
// cookies that the Client keeps for later calls.
func Login(ctx context.Context, username, password string, opts ...Option) (*Client, error) {
	cfg := newConfig(opts)
	t := newTransport(cfg)

	form := url.Values{"username": {username}, "password": {password}}.Encode()
	steps := []struct {
		method, path, body string
		want               int
	}{
		{"GET", "/logowanie", "", http.StatusOK},
		{"GET", "/api/profile/accounts", "", http.StatusUnauthorized},
		{"GET", "/api/open/analytical-tools-settings?companyName=kuchniavikinga&host=" + url.QueryEscape(t.host()), "", http.StatusOK},
		{"POST", "/api/auth/login", form, http.StatusOK},
	}
	for _, s := range steps {
		var body *strings.Reader
		if s.body != "" {
			body = strings.NewReader(s.body)
		}
		if err := t.send(ctx, s.method, s.path, body, s.want); err != nil {
			return nil, fmt.Errorf("login: %w", err)
		}
	}
	return &Client{t: t}, nil
}

// Logout ends the session. The Client must not be used afterwards.
func (c *Client) Logout(ctx context.Context) error {
	return c.t.send(ctx, "POST", "/api/auth/logout", nil, http.StatusOK)
}
