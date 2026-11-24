package kuchniavikinga

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	basePath   = "/api/company/customer/order/7/deliveries/123000000/delivery-meals/5/switch"
	deliveryID = 123000000
)

// fakePanel emulates the login flow and the endpoints; csrf collects the
// X-XSRF-TOKEN header seen on PUT requests.
func fakePanel(t *testing.T) (srv *httptest.Server, csrf *[]string) {
	t.Helper()
	csrf = new([]string)
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
		w.Write([]byte(`[7]`))
	})
	mux.HandleFunc("GET /api/company/customer/order/7", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"dateFrom":"2026-01-01","dateTo":"2026-01-31","deliveries":[{"date":"2026-01-02","deleted":false,"deliveryId":1.23E+8}]}`))
	})
	mux.HandleFunc("GET /api/company/general/menus/delivery/123000000/new", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"deliveryMenuMeal":[{"deliveryMealId":5,"mealName":"Śniadanie","menuMealName":"Owsianka","dietCaloriesMealId":9,"nutrition":{"calories":350,"caloriesText":"350 kcal"},"switchable":true,"ingredients":[{"name":"owies","major":true}]}]}`))
	})
	mux.HandleFunc("GET "+basePath+"/new", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"mealChangeOptions":[
			{"menuMealDetails":{"menuMealName":"Zupa","dietCaloriesMealId":11},"dietOptionName":"Low carb","canBeChanged":true,"reviewSummary":{"score":4.5,"number":12}},
			{"menuMealDetails":{"menuMealName":"Pierogi","dietCaloriesMealId":12},"canBeChanged":false,"reviewSummary":null}]}`))
	})
	mux.HandleFunc("PUT "+basePath, func(w http.ResponseWriter, r *http.Request) {
		*csrf = append(*csrf, r.Header.Get("X-XSRF-TOKEN"))
		if r.URL.Query().Get("dietCaloriesMealId") != "11" {
			w.WriteHeader(400)
			return
		}
		w.Write([]byte(`{"deliveryMealId":5,"dietCaloriesMealId":11,"seen":"SWITCHED"}`))
	})
	mux.HandleFunc("POST /api/auth/logout", func(w http.ResponseWriter, r *http.Request) {})
	return httptest.NewServer(mux), csrf
}

func TestFullFlow(t *testing.T) {
	srv, csrf := fakePanel(t)
	defer srv.Close()
	ctx := context.Background()

	c, err := Login(ctx, "u", "p", WithBaseURL(srv.URL))
	if err != nil {
		t.Fatal(err)
	}
	orders, err := c.Orders(ctx)
	if err != nil || len(orders) != 1 {
		t.Fatalf("Orders = %v, %v", orders, err)
	}
	o := orders[0]
	if o.ID != 7 || !o.From.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) || len(o.Deliveries) != 1 {
		t.Fatalf("order = %+v", o)
	}
	d := o.Deliveries[0]
	if d.ID != ID(deliveryID) || d.Cancelled || d.Date.Day() != 2 {
		t.Fatalf("scientific-notation delivery = %+v", d)
	}

	meals, err := d.Meals(ctx)
	if err != nil || len(meals) != 1 {
		t.Fatalf("Meals = %v, %v", meals, err)
	}
	m := meals[0]
	if m.Slot != "Śniadanie" || m.Name != "Owsianka" || m.Nutrition.Calories != 350 || !m.Switchable || !m.Dish.Ingredients[0].Major {
		t.Fatalf("meal = %+v", m)
	}

	alts, err := m.Alternatives(ctx)
	if err != nil || len(alts) != 2 {
		t.Fatalf("Alternatives = %v, %v", alts, err)
	}
	if alts[0].Rating == nil || alts[0].Rating.Votes != 12 || alts[0].Diet != "Low carb" || !alts[0].Allowed {
		t.Fatalf("alt[0] = %+v", alts[0])
	}
	if alts[1].Rating != nil || alts[1].Allowed {
		t.Fatalf("alt[1] = %+v", alts[1])
	}

	if err := m.SwitchTo(ctx, alts[1]); !errors.Is(err, ErrNotSwitchable) {
		t.Fatalf("SwitchTo(disallowed) = %v, want ErrNotSwitchable", err)
	}
	if err := m.SwitchTo(ctx, alts[0]); err != nil {
		t.Fatal(err)
	}
	if m.Dish.Name != "Zupa" {
		t.Fatalf("meal not updated after switch: %+v", m.Dish)
	}
	if len(*csrf) != 1 || (*csrf)[0] != "tok2" {
		t.Fatalf("CSRF header on PUT = %v, want rotated token tok2", *csrf)
	}
	if err := c.Logout(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestLoginFailureIsStatusError(t *testing.T) {
	srv, _ := fakePanel(t)
	defer srv.Close()
	_, err := Login(context.Background(), "u", "bad", WithBaseURL(srv.URL))
	var se *StatusError
	if !errors.As(err, &se) || se.Got != 401 || se.Want != 200 {
		t.Fatalf("err = %v, want *StatusError 401", err)
	}
}

func TestDebugOutput(t *testing.T) {
	srv, _ := fakePanel(t)
	defer srv.Close()
	var sb strings.Builder
	if _, err := Login(context.Background(), "u", "p", WithBaseURL(srv.URL), WithDebug(&sb)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sb.String(), "=== HTTP REQUEST ===") {
		t.Fatal("no debug dump written")
	}
}
