package lobby

import (
	"context"

	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type mockRunner struct {
	RunFunc func(ctx context.Context, in <-chan message.Message) <-chan message.Message
}

func (m *mockRunner) Run(ctx context.Context, in <-chan message.Message) <-chan message.Message {
	return m.RunFunc(ctx, in)
}
