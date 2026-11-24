package kuchniavikinga

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
)

// ID identifies an order, delivery, meal or dish. The panel sometimes sends
// ids in scientific notation (1.23E+8); ID accepts both forms.
type ID int64

// UnmarshalJSON accepts plain and scientific-notation numbers.
func (i *ID) UnmarshalJSON(b []byte) error {
	f, err := strconv.ParseFloat(string(b), 64)
	*i = ID(f)
	return err
}

// Order is a customer order: a period of daily deliveries.
type Order struct {
	ID ID `json:"-"`
	// From and To bound the order (midnight UTC; only the date is meaningful).
	From       time.Time   `json:"-"`
	To         time.Time   `json:"-"`
	Deliveries []*Delivery `json:"deliveries"`
}

// Delivery is one day's delivery within an Order.
type Delivery struct {
	ID ID `json:"deliveryId"`
	// Date is the delivery day (midnight UTC; only the date is meaningful).
	Date time.Time `json:"-"`
	// Cancelled is true for deliveries that were removed from the order.
	Cancelled bool `json:"deleted"`

	c       *Client
	orderID ID
}

// Orders returns the customer's active orders with their deliveries.
//
//	GET /api/company/customer/order/active-ids, then GET .../order/{id} for each
func (c *Client) Orders(ctx context.Context) ([]*Order, error) {
	ids, err := get[[]ID](ctx, c.t, "GET", "/api/company/customer/order/active-ids")
	if err != nil {
		return nil, err
	}
	orders := make([]*Order, len(ids))
	for i, id := range ids {
		if orders[i], err = c.Order(ctx, id); err != nil {
			return nil, err
		}
	}
	return orders, nil
}

// Order returns one order by id.
//
//	GET /api/company/customer/order/{id}
func (c *Client) Order(ctx context.Context, id ID) (*Order, error) {
	o, err := get[*Order](ctx, c.t, "GET", fmt.Sprintf("/api/company/customer/order/%d", id))
	if err != nil {
		return nil, err
	}
	o.ID = id
	for _, d := range o.Deliveries {
		d.c, d.orderID = c, id
	}
	return o, nil
}

// UnmarshalJSON decodes the panel's "YYYY-MM-DD" dates into From and To.
func (o *Order) UnmarshalJSON(b []byte) error {
	type plain Order
	aux := struct {
		*plain
		From string `json:"dateFrom"`
		To   string `json:"dateTo"`
	}{plain: (*plain)(o)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	var err1, err2 error
	o.From, err1 = time.Parse(time.DateOnly, aux.From)
	o.To, err2 = time.Parse(time.DateOnly, aux.To)
	return errors.Join(err1, err2)
}

// UnmarshalJSON decodes the panel's "YYYY-MM-DD" date into Date.
func (d *Delivery) UnmarshalJSON(b []byte) (err error) {
	type plain Delivery
	aux := struct {
		*plain
		Date string `json:"date"`
	}{plain: (*plain)(d)}
	if err = json.Unmarshal(b, &aux); err != nil {
		return err
	}
	d.Date, err = time.Parse(time.DateOnly, aux.Date)
	return err
}
