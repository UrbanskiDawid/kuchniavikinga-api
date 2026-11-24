# kuchniawikinga-api

Unofficial Go client for the customer API of [panel.kuchniavikinga.pl](https://panel.kuchniavikinga.pl)
(Kuchnia Vikinga diet catering). Pure API access: no scoring, no UI, no config.
Standard library only, zero dependencies.

> The API is private and may change without notice. This client replays what
> the web panel does in a browser. Not affiliated with Kuchnia Vikinga.

## Install

```sh
go get github.com/UrbanskiDawid/kuchniawikinga-api
```

```go
import kuchniavikinga "github.com/UrbanskiDawid/kuchniawikinga-api"
```

## Quick start

```go
ctx := context.Background()
c := kuchniavikinga.New(kuchniavikinga.Options{})
if err := c.Open(ctx, kuchniavikinga.Auth{Username: user, Password: pass}); err != nil {
    log.Fatal(err)
}
defer c.Close(ctx)

orders, _ := c.ActiveOrders(ctx)            // []OrderID
order, _ := c.Order(ctx, orders[0])         // deliveries per day
d := order.Deliveries[0]
menu, _ := c.Menu(ctx, d.ID())              // meals of that delivery
m := menu.DeliveryMenuMeal[0]
opts, _ := c.SwitchOptions(ctx, orders[0], d.ID(), m.DeliveryMealID)
// modifies your real order:
_, err := c.Switch(ctx, orders[0], d.ID(), m.DeliveryMealID, opts[0].MenuMealDetails.DietCaloriesMealID)
```

## Endpoints

| Method | Go | HTTP |
|---|---|---|
| `Open` | log in | `GET /logowanie`, `GET /api/profile/accounts` (401 expected), `GET /api/open/analytical-tools-settings`, `POST /api/auth/login` |
| `Close` | log out | `POST /api/auth/logout` |
| `ActiveOrders` | active order ids | `GET /api/company/customer/order/active-ids` |
| `Order` | order with deliveries | `GET /api/company/customer/order/{id}` |
| `Menu` | meals of a delivery | `GET /api/company/general/menus/delivery/{id}/new` |
| `SwitchOptions` | possible replacements | `GET .../order/{o}/deliveries/{d}/delivery-meals/{m}/switch/new` |
| `Switch` | replace a meal | `PUT .../switch?amount=1&dietCaloriesMealId={id}` |

## Notes

- **CSRF:** the panel uses Spring CSRF. Non-GET requests automatically send the
  `X-XSRF-TOKEN` header taken from the `XSRF-TOKEN` cookie.
- **Delivery ids** can arrive in scientific notation; use `Delivery.ID()`.
- **Errors:** unexpected HTTP statuses are returned as `*StatusError`
  (use `errors.As`).
- **Customisation:** `Options.HTTPClient` lets you plug in a custom
  `Transport` (caching, proxy, metrics); `Options.BaseURL` targets a test
  server; `Options.DebugOutput` dumps traffic (**including your password**).
- Full reference: `go doc -all github.com/UrbanskiDawid/kuchniawikinga-api`.

## License

MIT
