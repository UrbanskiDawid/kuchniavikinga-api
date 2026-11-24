package kuchniavikinga_test

import (
	"context"
	"fmt"

	kuchniavikinga "github.com/UrbanskiDawid/kuchniawikinga-api"
)

// List today's-order meals and their switch options (compile-checked only).
func Example() {
	ctx := context.Background()
	c := kuchniavikinga.New(kuchniavikinga.Options{})
	if err := c.Open(ctx, kuchniavikinga.Auth{Username: "me@example.com", Password: "secret"}); err != nil {
		panic(err)
	}
	defer c.Close(ctx)

	orders, _ := c.ActiveOrders(ctx)
	order, _ := c.Order(ctx, orders[0])
	delivery := order.Deliveries[0]
	menu, _ := c.Menu(ctx, delivery.ID())
	for _, meal := range menu.DeliveryMenuMeal {
		if !meal.Switchable {
			continue
		}
		options, _ := c.SwitchOptions(ctx, orders[0], delivery.ID(), meal.DeliveryMealID)
		for _, o := range options {
			fmt.Println(meal.MenuMealName, "->", o.MenuMealDetails.MenuMealName)
		}
	}
}
