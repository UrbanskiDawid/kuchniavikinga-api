package kuchniavikinga

import (
	"context"
	"fmt"
)

// Identifier types. They are distinct types so they cannot be mixed up by accident.
type (
	// OrderID identifies an order (a subscription period with many deliveries).
	OrderID int64
	// DeliveryID identifies one delivery (one day's boxes) within an order.
	DeliveryID int64
	// DeliveryMealID identifies one meal slot (e.g. breakfast) inside a delivery.
	DeliveryMealID int64
	// DietCaloriesMealID identifies a concrete dish in a given diet/calorie
	// variant; it is what Switch takes as its target.
	DietCaloriesMealID int
)

// Delivery is one day's delivery inside an Order.
type Delivery struct {
	// Date is the delivery day in YYYY-MM-DD form.
	Date string `json:"date"`
	// Deleted is true for cancelled deliveries.
	Deleted bool `json:"deleted"`
	// DeliveryIDFloat is the raw id. The API sometimes sends it in scientific
	// notation, so it is decoded as float64; use ID instead.
	DeliveryIDFloat float64 `json:"deliveryId"`
}

// ID returns the delivery id as a DeliveryID.
func (d Delivery) ID() DeliveryID { return DeliveryID(d.DeliveryIDFloat) }

// Order is a customer order with its deliveries.
type Order struct {
	// DateFrom and DateTo bound the order, in YYYY-MM-DD form.
	DateFrom   string     `json:"dateFrom"`
	DateTo     string     `json:"dateTo"`
	Deliveries []Delivery `json:"deliveries"`
}

// ActiveOrders returns the ids of the customer's active orders.
//
//	GET /api/company/customer/order/active-ids
func (c *Client) ActiveOrders(ctx context.Context) ([]OrderID, error) {
	ids, err := requestJSON[[]OrderID](ctx, c, "GET", "/api/company/customer/order/active-ids")
	if err != nil {
		return nil, err
	}
	return *ids, nil
}

// Order returns an order with its deliveries.
//
//	GET /api/company/customer/order/{orderId}
func (c *Client) Order(ctx context.Context, id OrderID) (*Order, error) {
	return requestJSON[Order](ctx, c, "GET", fmt.Sprintf("/api/company/customer/order/%d", id))
}
