package game

import "context"

type mockWordValidator struct {
	ValidateFunc func(word string) bool
}

func (wc mockWordValidator) Validate(word string) bool {
	return wc.ValidateFunc(word)
}

type mockUserDao struct {
	UpdatePointsIncrementFunc func(ctx context.Context, userPoints map[string]int) error
}

func (ud mockUserDao) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	return ud.UpdatePointsIncrementFunc(ctx, userPoints)
}
