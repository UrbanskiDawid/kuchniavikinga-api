package kuchniavikinga

import (
	"context"
	"fmt"
)

// MealOption is one meal that a delivery meal can be switched to.
type MealOption struct {
	MenuMealDetails MenuMealDetails `json:"menuMealDetails"`
	// DietOptionName is the diet variant, e.g. "Low carb" or "Fodmap".
	DietOptionName string `json:"dietOptionName"`
	// CanBeChanged is false when switching to this option is not allowed.
	CanBeChanged bool `json:"canBeChanged"`
	// ReviewSummary is nil when there are no reviews.
	ReviewSummary *ReviewSummary `json:"reviewSummary"`
}

// MenuMealDetails describes a dish offered as a switch target.
type MenuMealDetails struct {
	MenuMealName       string             `json:"menuMealName"`
	DietCaloriesMealID DietCaloriesMealID `json:"dietCaloriesMealId"`
	Nutrition          Nutrition          `json:"nutrition"`
	Ingredients        []Ingredient       `json:"ingredients"`
}

// ReviewSummary is the aggregate of other customers' reviews of a dish.
type ReviewSummary struct {
	Score  float64 `json:"score"`
	Number int     `json:"number"`
}

// SwitchResponse is the result of a successful Switch.
type SwitchResponse struct {
	DeliveryMealID     DeliveryMealID     `json:"deliveryMealId"`
	DietCaloriesMealID DietCaloriesMealID `json:"dietCaloriesMealId"`
	// Seen is "SWITCHED" on success.
	Seen string `json:"seen"`
}

func switchPath(order OrderID, delivery DeliveryID, meal DeliveryMealID) string {
	return fmt.Sprintf("/api/company/customer/order/%d/deliveries/%d/delivery-meals/%d/switch", order, delivery, meal)
}

// SwitchOptions lists the meals that a delivery meal can be switched to.
//
//	GET /api/company/customer/order/{order}/deliveries/{delivery}/delivery-meals/{meal}/switch/new
func (c *Client) SwitchOptions(ctx context.Context, order OrderID, delivery DeliveryID, meal DeliveryMealID) ([]MealOption, error) {
	resp, err := requestJSON[struct {
		MealChangeOptions []MealOption `json:"mealChangeOptions"`
	}](ctx, c, "GET", switchPath(order, delivery, meal)+"/new")
	if err != nil {
		return nil, err
	}
	return resp.MealChangeOptions, nil
}

// Switch replaces a delivery meal with the given option. This modifies the
// customer's real order. It sends the CSRF header automatically.
//
//	PUT /api/company/customer/order/{order}/deliveries/{delivery}/delivery-meals/{meal}/switch?amount=1&dietCaloriesMealId={to}
func (c *Client) Switch(ctx context.Context, order OrderID, delivery DeliveryID, meal DeliveryMealID, to DietCaloriesMealID) (*SwitchResponse, error) {
	path := fmt.Sprintf("%s?amount=1&dietCaloriesMealId=%d", switchPath(order, delivery, meal), to)
	return requestJSON[SwitchResponse](ctx, c, "PUT", path)
}
