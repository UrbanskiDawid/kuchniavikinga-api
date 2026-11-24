package kuchniavikinga

import (
	"context"
	"fmt"
	"time"
)

// Vote is your thumbs-up or thumbs-down for a meal.
type Vote int

const (
	VoteNone Vote = iota // not voted
	VoteUp               // thumbs-up
	VoteDown             // thumbs-down
)

// String returns "up", "down" or "none".
func (v Vote) String() string {
	switch v {
	case VoteUp:
		return "up"
	case VoteDown:
		return "down"
	}
	return "none"
}

// The panel stores a vote as a score: 1 for up, 0 for down.
func voteFromScore(score int) Vote {
	switch score {
	case 1:
		return VoteUp
	case 0:
		return VoteDown
	}
	return VoteNone
}

// Vote returns your vote as of when the meal was fetched (no network call).
func (m *Meal) Vote() Vote { return m.vote }

// SetVote stores a vote on the panel, replacing any earlier one, and updates
// m.Vote. It changes real data; the panel only accepts votes on current meals.
// The panel has no way to withdraw a vote, so VoteNone returns ErrInvalidVote.
//
//	POST /api/company/customer/review {"reviewId":{"menuMealId":N},"score":1|0,"text":""}
func (m *Meal) SetVote(ctx context.Context, v Vote) error {
	var score int
	switch v {
	case VoteUp:
		score = 1
	case VoteDown:
		score = 0
	default:
		return ErrInvalidVote
	}
	body := map[string]any{
		"reviewId": map[string]any{"menuMealId": m.MenuID},
		"score":    score,
		"text":     "",
	}
	if err := m.c.t.sendJSON(ctx, "POST", "/api/company/customer/review", body); err != nil {
		return err
	}
	m.vote = v
	return nil
}

// MyVotes returns the meals you have voted on among the deliveries of the
// given day (only the date part of day is used).
func (c *Client) MyVotes(ctx context.Context, day time.Time) ([]*Meal, error) {
	orders, err := c.Orders(ctx)
	if err != nil {
		return nil, err
	}
	want := day.Format(time.DateOnly)
	var voted []*Meal
	for _, o := range orders {
		for _, d := range o.Deliveries {
			if d.Cancelled || d.Date.Format(time.DateOnly) != want {
				continue
			}
			meals, err := d.Meals(ctx)
			if err != nil {
				return nil, fmt.Errorf("delivery %d: %w", d.ID, err)
			}
			for _, m := range meals {
				if m.Vote() != VoteNone {
					voted = append(voted, m)
				}
			}
		}
	}
	return voted, nil
}
