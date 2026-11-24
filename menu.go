package kuchniavikinga

import (
	"context"
	"fmt"
)

// DeliveryMeal is a meal currently assigned to a delivery.
type DeliveryMeal struct {
	DeliveryMealID DeliveryMealID `json:"deliveryMealId"`
	// MealName is the slot name, e.g. "Śniadanie" (breakfast).
	MealName string `json:"mealName"`
	// MenuMealName is the dish name.
	MenuMealName       string             `json:"menuMealName"`
	DietCaloriesMealID DietCaloriesMealID `json:"dietCaloriesMealId"`
	Nutrition          Nutrition          `json:"nutrition"`
	Ingredients        []Ingredient       `json:"ingredients"`
	// Switchable reports whether the meal can be replaced (see SwitchOptions).
	Switchable bool `json:"switchable"`
}

// Nutrition holds the energy value of a dish.
type Nutrition struct {
	Calories     float64 `json:"calories"`
	CaloriesText string  `json:"caloriesText"`
}

// Ingredient is one ingredient of a dish.
type Ingredient struct {
	Name string `json:"name"`
	// Major marks the main ingredients.
	Major bool `json:"major"`
}

// Menu is the list of meals of one delivery.
type Menu struct {
	DeliveryMenuMeal []DeliveryMeal `json:"deliveryMenuMeal"`
}

// Menu returns the meals of a delivery.
//
//	GET /api/company/general/menus/delivery/{deliveryId}/new
func (c *Client) Menu(ctx context.Context, id DeliveryID) (*Menu, error) {
	return requestJSON[Menu](ctx, c, "GET", fmt.Sprintf("/api/company/general/menus/delivery/%d/new", id))
}
