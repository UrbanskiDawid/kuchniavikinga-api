package kuchniavikinga_test

import (
	"context"
	"fmt"
	"time"

	kuchniavikinga "github.com/UrbanskiDawid/kuchniavikinga-api"
)

// Print what each switchable meal of the first delivery could be swapped for
// (compile-checked only; it needs a real account).
func Example() {
	ctx := context.Background()
	c, err := kuchniavikinga.Login(ctx, "me@example.com", "secret")
	if err != nil {
		panic(err)
	}
	defer c.Logout(ctx)

	orders, _ := c.Orders(ctx)
	meals, _ := orders[0].Deliveries[0].Meals(ctx)
	for _, m := range meals {
		if !m.Switchable {
			continue
		}
		alts, _ := m.Alternatives(ctx)
		for _, a := range alts {
			fmt.Printf("%s: %s -> %s\n", m.Slot, m.Name, a.Dish.Name)
		}
	}
}

// Thumbs-up every meal of the first delivery, then list today's votes
// (compile-checked only; it changes your real data).
func Example_votes() {
	ctx := context.Background()
	c, err := kuchniavikinga.Login(ctx, "me@example.com", "secret")
	if err != nil {
		panic(err)
	}
	defer c.Logout(ctx)

	orders, _ := c.Orders(ctx)
	meals, _ := orders[0].Deliveries[0].Meals(ctx)
	for _, m := range meals {
		_ = m.SetVote(ctx, kuchniavikinga.VoteUp)
	}
	voted, _ := c.MyVotes(ctx, time.Now())
	for _, m := range voted {
		fmt.Println(m.Name, m.Vote())
	}
}
