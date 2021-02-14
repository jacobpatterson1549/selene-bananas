package game

import "context"

type mockWordChecker struct {
	CheckFunc func(word string) bool
}

func (wc mockWordChecker) Check(word string) bool {
	return wc.CheckFunc(word)
}

type mockUserDao struct {
	UpdatePointsIncrementFunc func(ctx context.Context, userPoints map[string]int) error
}

func (ud mockUserDao) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	return ud.UpdatePointsIncrementFunc(ctx, userPoints)
}
