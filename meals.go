package kuchniavikinga

import (
	"context"
	"fmt"
)

// Dish is a dish on the menu.
type Dish struct {
	// ID identifies the dish in its diet/calorie variant.
	ID   int64
	Name string
	// Calories in kcal; CaloriesText is the panel's display string.
	Calories     float64
	CaloriesText string
	Ingredients  []Ingredient
}

// Ingredient is one ingredient of a Dish.
type Ingredient struct {
	Name string
	// Major marks the main ingredients.
	Major bool
}

// Meal is one slot of a delivery (breakfast, lunch, ...) and the dish
// currently assigned to it.
type Meal struct {
	ID int64
	// Slot is the meal's name in the day, e.g. "Śniadanie" (breakfast).
	Slot string
	// Dish is what you will receive. SwitchTo updates it.
	Dish Dish
	// Switchable reports whether the panel lets you replace this meal.
	Switchable bool

	c                   *Client
	orderID, deliveryID int64
}

// Alternative is a dish that a Meal can be switched to.
type Alternative struct {
	Dish Dish
	// Diet is the diet variant, e.g. "Low carb" or "Fodmap".
	Diet string
	// Allowed is false when the panel does not permit switching to it.
	Allowed bool
	// Rating is nil when nobody has reviewed the dish.
	Rating *Rating
}

// Rating is the aggregate of other customers' reviews of a dish.
type Rating struct {
	Score float64
	Votes int
}

// Meals returns the meals of this delivery.
//
//	GET /api/company/general/menus/delivery/{id}/new
func (d *Delivery) Meals(ctx context.Context) ([]*Meal, error) {
	var w wireMenu
	if err := d.c.t.getJSON(ctx, "GET", fmt.Sprintf("/api/company/general/menus/delivery/%d/new", d.ID), &w); err != nil {
		return nil, err
	}
	meals := make([]*Meal, 0, len(w.Meals))
	for _, m := range w.Meals {
		meals = append(meals, &Meal{
			ID:         m.DeliveryMealID,
			Slot:       m.MealName,
			Dish:       dish(m.DietCaloriesMealID, m.MenuMealName, m.Nutrition, m.Ingredients),
			Switchable: m.Switchable,
			c:          d.c, orderID: d.orderID, deliveryID: d.ID,
		})
	}
	return meals, nil
}

func (m *Meal) path() string {
	return fmt.Sprintf("/api/company/customer/order/%d/deliveries/%d/delivery-meals/%d/switch", m.orderID, m.deliveryID, m.ID)
}

// Alternatives lists the dishes this meal can be switched to.
//
//	GET .../order/{order}/deliveries/{delivery}/delivery-meals/{meal}/switch/new
func (m *Meal) Alternatives(ctx context.Context) ([]Alternative, error) {
	var w wireOptions
	if err := m.c.t.getJSON(ctx, "GET", m.path()+"/new", &w); err != nil {
		return nil, err
	}
	alts := make([]Alternative, 0, len(w.MealChangeOptions))
	for _, o := range w.MealChangeOptions {
		d := o.MenuMealDetails
		a := Alternative{
			Dish:    dish(d.DietCaloriesMealID, d.MenuMealName, d.Nutrition, d.Ingredients),
			Diet:    o.DietOptionName,
			Allowed: o.CanBeChanged,
		}
		if o.ReviewSummary != nil {
			a.Rating = &Rating{Score: o.ReviewSummary.Score, Votes: o.ReviewSummary.Number}
		}
		alts = append(alts, a)
	}
	return alts, nil
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
	var resp struct {
		Seen string `json:"seen"`
	}
	path := fmt.Sprintf("%s?amount=1&dietCaloriesMealId=%d", m.path(), alt.Dish.ID)
	if err := m.c.t.getJSON(ctx, "PUT", path, &resp); err != nil {
		return err
	}
	if resp.Seen != "SWITCHED" {
		return fmt.Errorf("switch: unexpected response %q", resp.Seen)
	}
	m.Dish = alt.Dish
	return nil
}

func dish(id int64, name string, n wireNutrition, ing []wireIngredient) Dish {
	d := Dish{ID: id, Name: name, Calories: n.Calories, CaloriesText: n.CaloriesText}
	for _, i := range ing {
		d.Ingredients = append(d.Ingredients, Ingredient{Name: i.Name, Major: i.Major})
	}
	return d
}
