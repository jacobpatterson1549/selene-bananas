package lobby

import (
	"context"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type mockRunner func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message

func (m mockRunner) Run(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
	return m(ctx, wg, in)
}
