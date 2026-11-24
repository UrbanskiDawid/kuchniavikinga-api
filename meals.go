package kuchniavikinga

import (
	"context"
	"encoding/json"
	"fmt"
)

// Dish is a dish on the menu.
type Dish struct {
	// ID identifies the dish in its diet/calorie variant.
	ID   ID     `json:"dietCaloriesMealId"`
	Name string `json:"menuMealName"` // dish name, in Polish

	Nutrition   Nutrition    `json:"nutrition"`
	Ingredients []Ingredient `json:"ingredients"`
}

// Nutrition is the energy value of a dish.
type Nutrition struct {
	Calories float64 `json:"calories"` // kcal
	Text     string  `json:"caloriesText"`
}

// Ingredient is one ingredient of a Dish.
type Ingredient struct {
	Name  string `json:"name"`
	Major bool   `json:"major"` // marks the main ingredients
}

// Meal is one slot of a delivery (breakfast, lunch, ...) and the dish
// currently assigned to it. The Dish fields are promoted: m.Name is the
// dish name.
type Meal struct {
	ID ID `json:"deliveryMealId"`
	// Slot is the meal's name in the day, e.g. "Śniadanie" (breakfast).
	Slot string `json:"mealName"`
	// Dish is what you will receive. SwitchTo updates it.
	Dish
	// Switchable reports whether the panel lets you replace this meal.
	Switchable bool `json:"switchable"`

	// MenuID identifies the menu entry; it is the key of votes (see SetVote).
	MenuID ID `json:"menuMealId"`

	vote                Vote
	c                   *Client
	orderID, deliveryID ID
}

// UnmarshalJSON decodes a meal and the vote recorded on it (see Vote).
func (m *Meal) UnmarshalJSON(b []byte) error {
	type plain Meal
	aux := struct {
		*plain
		Review *struct {
			Score *int `json:"score"`
		} `json:"review"`
	}{plain: (*plain)(m)}
	if err := json.Unmarshal(b, &aux); err != nil {
		return err
	}
	m.vote = VoteNone
	if aux.Review != nil && aux.Review.Score != nil {
		m.vote = voteFromScore(*aux.Review.Score)
	}
	return nil
}

// Alternative is a dish that a Meal can be switched to.
type Alternative struct {
	Dish Dish `json:"menuMealDetails"`
	// Diet is the diet variant, e.g. "Low carb" or "Fodmap".
	Diet string `json:"dietOptionName"`
	// Allowed is false when the panel does not permit switching to it.
	Allowed bool `json:"canBeChanged"`
	// Rating is nil when nobody has reviewed the dish.
	Rating *Rating `json:"reviewSummary"`
}

// Rating is the aggregate of other customers' reviews of a dish.
type Rating struct {
	Score float64 `json:"score"` // average score
	Votes int     `json:"number"`
}

// Meals returns the meals of this delivery.
//
//	GET /api/company/general/menus/delivery/{id}/new
func (d *Delivery) Meals(ctx context.Context) ([]*Meal, error) {
	menu, err := get[struct {
		Meals []*Meal `json:"deliveryMenuMeal"`
	}](ctx, d.c.t, "GET", fmt.Sprintf("/api/company/general/menus/delivery/%d/new", d.ID))
	if err != nil {
		return nil, err
	}
	for _, m := range menu.Meals {
		m.c, m.orderID, m.deliveryID = d.c, d.orderID, d.ID
	}
	return menu.Meals, nil
}

func (m *Meal) path() string {
	return fmt.Sprintf("/api/company/customer/order/%d/deliveries/%d/delivery-meals/%d/switch", m.orderID, m.deliveryID, m.ID)
}

// Alternatives lists the dishes this meal can be switched to.
//
//	GET .../order/{order}/deliveries/{delivery}/delivery-meals/{meal}/switch/new
func (m *Meal) Alternatives(ctx context.Context) ([]Alternative, error) {
	r, err := get[struct {
		Options []Alternative `json:"mealChangeOptions"`
	}](ctx, m.c.t, "GET", m.path()+"/new")
	return r.Options, err
}

// SwitchTo replaces this meal with the alternative and updates m.Dish. It
// changes the customer's real order. It returns ErrNotSwitchable if the
// alternative is not Allowed.
//
//	PUT .../switch?amount=1&dietCaloriesMealId={alt.Dish.ID}
func (m *Meal) SwitchTo(ctx context.Context, alt Alternative) error {
	if !alt.Allowed {
		return ErrNotSwitchable
	}
	r, err := get[struct{ Seen string }](ctx, m.c.t, "PUT", fmt.Sprintf("%s?amount=1&dietCaloriesMealId=%d", m.path(), alt.Dish.ID))
	if err != nil {
		return err
	}
	if r.Seen != "SWITCHED" {
		return fmt.Errorf("switch: unexpected response %q", r.Seen)
	}
	m.Dish = alt.Dish
	return nil
}
