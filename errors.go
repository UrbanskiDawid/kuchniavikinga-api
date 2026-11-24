package kuchniavikinga

import (
	"errors"
	"fmt"
	"net/http"
)

// ErrNotSwitchable is returned by Meal.SwitchTo when the panel does not allow
// the chosen alternative (see Alternative.Allowed).
var ErrNotSwitchable = errors.New("kuchniavikinga: alternative cannot be switched to")

// StatusError is returned when the panel answers with an unexpected HTTP
// status. Use errors.As to inspect it, e.g. to detect a 401 or 403.
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
