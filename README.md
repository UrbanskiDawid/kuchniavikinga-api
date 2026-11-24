# kuchniavikinga-api

[![Go Reference](https://pkg.go.dev/badge/github.com/UrbanskiDawid/kuchniavikinga-api.svg)](https://pkg.go.dev/github.com/UrbanskiDawid/kuchniavikinga-api)

Unofficial Go client for the customer API of [panel.kuchniavikinga.pl](https://panel.kuchniavikinga.pl)
(Kuchnia Vikinga diet catering). Pure API access: no scoring, no UI, no config.
Standard library only, zero dependencies.

> The API is private and may change without notice. This client replays what
> the web panel does in a browser. Not affiliated with Kuchnia Vikinga.

## Install

```sh
go get github.com/UrbanskiDawid/kuchniavikinga-api
```

```go
import kuchniavikinga "github.com/UrbanskiDawid/kuchniavikinga-api"
```

## Quick start

```go
ctx := context.Background()
c, err := kuchniavikinga.Login(ctx, user, pass)
if err != nil {
    log.Fatal(err)
}
defer c.Logout(ctx)

orders, _ := c.Orders(ctx)                  // active orders, with deliveries
d := orders[0].Deliveries[0]                // one per delivery day
meals, _ := d.Meals(ctx)                    // breakfast, lunch, ...
alts, _ := meals[0].Alternatives(ctx)       // what it can be swapped for
err = meals[0].SwitchTo(ctx, alts[0])       // modifies your real order

v := meals[0].Vote()                        // VoteUp, VoteDown or VoteNone
err = meals[0].SetVote(ctx, kuchniavikinga.VoteUp) // 👍 stored on the panel
voted, _ := c.MyVotes(ctx, time.Now())      // meals you voted on today
```

## Model

You walk down from the account to a single meal; every level is a handle that
carries its own ids, so there are no id arguments to pass around.

| Type | Obtained by | Methods |
|---|---|---|
| `Client` | `Login` | `Orders`, `Order`, `MyVotes`, `Logout` |
| `Order` | `Client.Orders` | field `Deliveries` |
| `Delivery` | `Order.Deliveries` | `Meals` |
| `Meal` | `Delivery.Meals` | `Alternatives`, `SwitchTo`, `Vote`, `SetVote` |
| `Alternative` | `Meal.Alternatives` | plain data (`Dish`, `Diet`, `Allowed`, `Rating`) |

Options for `Login`: `WithBaseURL`, `WithTimeout`, `WithHTTPClient`, `WithDebug`.

## Endpoints

| Call | HTTP |
|---|---|
| `Login` | `GET /logowanie`, `GET /api/profile/accounts` (401 expected), `GET /api/open/analytical-tools-settings`, `POST /api/auth/login` |
| `Logout` | `POST /api/auth/logout` |
| `Orders` / `Order` | `GET /api/company/customer/order/active-ids`, `GET .../order/{id}` |
| `Delivery.Meals` | `GET /api/company/general/menus/delivery/{id}/new` |
| `Meal.Alternatives` | `GET .../order/{o}/deliveries/{d}/delivery-meals/{m}/switch/new` |
| `Meal.SwitchTo` | `PUT .../switch?amount=1&dietCaloriesMealId={id}` |
| `Meal.Vote` / `MyVotes` | read from `review` in the delivery menu response |
| `Meal.SetVote` | `POST /api/company/customer/review` (JSON; `score` 1 = up, 0 = down) |

## Notes

- **Votes:** 👍/👎 are stored on the panel (same as in the web app). `SetVote` overwrites
  an earlier vote and changes real data; the panel has no withdraw operation, so
  `VoteNone` is rejected with `ErrInvalidVote`. Only current meals can be voted on.
  Verified against the live panel (read, overwrite, re-read).
- **CSRF:** the panel uses Spring CSRF. Non-GET requests automatically send the
  `X-XSRF-TOKEN` header taken from the `XSRF-TOKEN` cookie.
- **Dates** are `time.Time` at midnight UTC; only the date part is meaningful.
- **Errors:** unexpected HTTP statuses are returned as `*StatusError`
  (use `errors.As`); `Meal.SwitchTo` returns `ErrNotSwitchable` for a disallowed alternative.
- **Customisation:** `WithHTTPClient` lets you plug in a custom
  `Transport` (caching, proxy, metrics); `WithBaseURL` targets a test
  server; `WithDebug` dumps traffic (**including your password**).
- Full reference: `go doc -all github.com/UrbanskiDawid/kuchniavikinga-api`.

## License

MIT

## Releasing

Push a tag (`git tag v0.3.0 && git push origin v0.3.0`). The `release`
workflow runs gofmt, `go vet` and `go test -race` first; the GitHub release is
created only if they pass.
