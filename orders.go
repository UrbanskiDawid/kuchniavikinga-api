package kuchniavikinga

import (
	"context"
	"fmt"
	"time"
)

// Order is a customer order: a period of daily deliveries.
type Order struct {
	ID int64
	// From and To bound the order (midnight UTC; only the date is meaningful).
	From, To   time.Time
	Deliveries []*Delivery
}

// Delivery is one day's delivery within an Order.
type Delivery struct {
	ID int64
	// Date is the delivery day (midnight UTC; only the date is meaningful).
	Date time.Time
	// Cancelled is true for deliveries that were removed from the order.
	Cancelled bool

	c       *Client
	orderID int64
}

// Orders returns the customer's active orders with their deliveries, oldest
// id first as the panel returns them.
//
//	GET /api/company/customer/order/active-ids, then GET /api/company/customer/order/{id} for each
func (c *Client) Orders(ctx context.Context) ([]*Order, error) {
	var ids []int64
	if err := c.t.getJSON(ctx, "GET", "/api/company/customer/order/active-ids", &ids); err != nil {
		return nil, err
	}
	orders := make([]*Order, 0, len(ids))
	for _, id := range ids {
		o, err := c.Order(ctx, id)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// Order returns one order by id.
//
//	GET /api/company/customer/order/{id}
func (c *Client) Order(ctx context.Context, id int64) (*Order, error) {
	var w wireOrder
	if err := c.t.getJSON(ctx, "GET", fmt.Sprintf("/api/company/customer/order/%d", id), &w); err != nil {
		return nil, err
	}
	o := &Order{ID: id}
	var err error
	if o.From, err = parseDate(w.DateFrom); err != nil {
		return nil, fmt.Errorf("order %d: dateFrom: %w", id, err)
	}
	if o.To, err = parseDate(w.DateTo); err != nil {
		return nil, fmt.Errorf("order %d: dateTo: %w", id, err)
	}
	for _, wd := range w.Deliveries {
		date, err := parseDate(wd.Date)
		if err != nil {
			return nil, fmt.Errorf("order %d: delivery date: %w", id, err)
		}
		o.Deliveries = append(o.Deliveries, &Delivery{
			ID: int64(wd.ID), Date: date, Cancelled: wd.Deleted, c: c, orderID: id,
		})
	}
	return o, nil
}

func parseDate(s string) (time.Time, error) { return time.Parse(time.DateOnly, s) }
