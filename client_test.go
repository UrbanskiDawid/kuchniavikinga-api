package kuchniavikinga

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakePanel emulates the login flow and a few endpoints.
func fakePanel(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var csrfOnPut []string
	mux := http.NewServeMux()
	mux.HandleFunc("GET /logowanie", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "XSRF-TOKEN", Value: "tok1", Path: "/"})
	})
	mux.HandleFunc("GET /api/profile/accounts", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) })
	mux.HandleFunc("GET /api/open/analytical-tools-settings", func(w http.ResponseWriter, r *http.Request) {})
	mux.HandleFunc("POST /api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.PostForm.Get("username") != "u" || r.PostForm.Get("password") != "p" {
			w.WriteHeader(401)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "XSRF-TOKEN", Value: "tok2", Path: "/"})
	})
	mux.HandleFunc("GET /api/company/customer/order/active-ids", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[7,8]`))
	})
	mux.HandleFunc("GET /api/company/customer/order/7", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"dateFrom":"2026-01-01","dateTo":"2026-01-31","deliveries":[{"date":"2026-01-02","deleted":false,"deliveryId":1.23E+8}]}`))
	})
	mux.HandleFunc("GET /api/company/general/menus/delivery/123000000/new", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"deliveryMenuMeal":[{"deliveryMealId":5,"mealName":"Śniadanie","menuMealName":"Owsianka","dietCaloriesMealId":9,"switchable":true,"ingredients":[{"name":"owies","major":true}]}]}`))
	})
	mux.HandleFunc("GET /api/company/customer/order/7/deliveries/123000000/delivery-meals/5/switch/new", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"mealChangeOptions":[{"menuMealDetails":{"menuMealName":"Zupa","dietCaloriesMealId":11},"dietOptionName":"Low carb","canBeChanged":true,"reviewSummary":null}]}`))
	})
	mux.HandleFunc("PUT /api/company/customer/order/7/deliveries/123000000/delivery-meals/5/switch", func(w http.ResponseWriter, r *http.Request) {
		csrfOnPut = append(csrfOnPut, r.Header.Get("X-XSRF-TOKEN"))
		if r.URL.Query().Get("dietCaloriesMealId") != "11" {
			w.WriteHeader(400)
			return
		}
		w.Write([]byte(`{"deliveryMealId":5,"dietCaloriesMealId":11,"seen":"SWITCHED"}`))
	})
	mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {})
	return httptest.NewServer(mux), &csrfOnPut
}

func TestFullFlow(t *testing.T) {
	srv, csrf := fakePanel(t)
	defer srv.Close()
	ctx := context.Background()
	c := New(Options{BaseURL: srv.URL})
	if err := c.Open(ctx, Auth{"u", "p"}); err != nil {
		t.Fatal(err)
	}
	ids, err := c.ActiveOrders(ctx)
	if err != nil || len(ids) != 2 || ids[0] != 7 {
		t.Fatalf("ActiveOrders = %v, %v", ids, err)
	}
	order, err := c.Order(ctx, 7)
	if err != nil || len(order.Deliveries) != 1 {
		t.Fatalf("Order = %+v, %v", order, err)
	}
	d := order.Deliveries[0]
	if d.ID() != 123000000 {
		t.Fatalf("scientific-notation id decoded as %d", d.ID())
	}
	menu, err := c.Menu(ctx, d.ID())
	if err != nil || menu.DeliveryMenuMeal[0].MenuMealName != "Owsianka" {
		t.Fatalf("Menu = %+v, %v", menu, err)
	}
	m := menu.DeliveryMenuMeal[0]
	opts, err := c.SwitchOptions(ctx, 7, d.ID(), m.DeliveryMealID)
	if err != nil || len(opts) != 1 || opts[0].ReviewSummary != nil || opts[0].MenuMealDetails.DietCaloriesMealID != 11 {
		t.Fatalf("SwitchOptions = %+v, %v", opts, err)
	}
	res, err := c.Switch(ctx, 7, d.ID(), m.DeliveryMealID, 11)
	if err != nil || res.Seen != "SWITCHED" {
		t.Fatalf("Switch = %+v, %v", res, err)
	}
	if len(*csrf) != 1 || (*csrf)[0] != "tok2" {
		t.Fatalf("CSRF header on PUT = %v, want rotated token tok2", *csrf)
	}
	if err := c.Close(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestLoginFailureIsStatusError(t *testing.T) {
	srv, _ := fakePanel(t)
	defer srv.Close()
	err := New(Options{BaseURL: srv.URL}).Open(context.Background(), Auth{"u", "bad"})
	var se *StatusError
	if !errors.As(err, &se) || se.Got != 401 || se.Want != 200 {
		t.Fatalf("err = %v, want *StatusError 401", err)
	}
}

func TestDebugOutput(t *testing.T) {
	srv, _ := fakePanel(t)
	defer srv.Close()
	var sb strings.Builder
	c := New(Options{BaseURL: srv.URL, DebugOutput: &sb})
	if err := c.Open(context.Background(), Auth{"u", "p"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "=== HTTP REQUEST ===") {
		t.Fatal("no debug dump written")
	}
}
