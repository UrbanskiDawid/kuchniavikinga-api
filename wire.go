package kuchniavikinga

// Wire types mirror the panel's JSON exactly and are converted to the public
// types in orders.go and meals.go. Keeping them separate lets the public API
// stay tidy while the panel's naming quirks stay contained here.

type wireOrder struct {
	DateFrom   string         `json:"dateFrom"`
	DateTo     string         `json:"dateTo"`
	Deliveries []wireDelivery `json:"deliveries"`
}

type wireDelivery struct {
	Date    string `json:"date"`
	Deleted bool   `json:"deleted"`
	// Sometimes sent in scientific notation, hence float64.
	ID float64 `json:"deliveryId"`
}

type wireNutrition struct {
	Calories     float64 `json:"calories"`
	CaloriesText string  `json:"caloriesText"`
}

type wireIngredient struct {
	Name  string `json:"name"`
	Major bool   `json:"major"`
}

type wireMenu struct {
	Meals []wireMeal `json:"deliveryMenuMeal"`
}

type wireMeal struct {
	DeliveryMealID     int64            `json:"deliveryMealId"`
	MealName           string           `json:"mealName"`
	MenuMealName       string           `json:"menuMealName"`
	DietCaloriesMealID int64            `json:"dietCaloriesMealId"`
	Nutrition          wireNutrition    `json:"nutrition"`
	Ingredients        []wireIngredient `json:"ingredients"`
	Switchable         bool             `json:"switchable"`
}

type wireOptions struct {
	MealChangeOptions []wireOption `json:"mealChangeOptions"`
}

type wireOption struct {
	MenuMealDetails struct {
		MenuMealName       string           `json:"menuMealName"`
		DietCaloriesMealID int64            `json:"dietCaloriesMealId"`
		Nutrition          wireNutrition    `json:"nutrition"`
		Ingredients        []wireIngredient `json:"ingredients"`
	} `json:"menuMealDetails"`
	DietOptionName string `json:"dietOptionName"`
	CanBeChanged   bool   `json:"canBeChanged"`
	ReviewSummary  *struct {
		Score  float64 `json:"score"`
		Number int     `json:"number"`
	} `json:"reviewSummary"`
}
