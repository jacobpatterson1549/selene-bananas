package game

import "context"

type mockUserDao struct {
	UpdatePointsIncrementFunc func(ctx context.Context, userPoints map[string]int) error
}

func (ud mockUserDao) UpdatePointsIncrement(ctx context.Context, userPoints map[string]int) error {
	return ud.UpdatePointsIncrementFunc(ctx, userPoints)
}
