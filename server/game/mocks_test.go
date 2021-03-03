package game

import "context"

type mockWordValidator func(word string) bool

func (m mockWordValidator) Validate(word string) bool {
	return m(word)
}

type mockUserDao struct {
	UpdatePointsIncrementFunc func(ctx context.Context, userPoints map[string]int) error
}

func (m mockUserDao) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	return m.UpdatePointsIncrementFunc(ctx, userPoints)
}
