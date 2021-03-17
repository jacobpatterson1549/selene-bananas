package lobby

import (
	"context"
	"sync"

	"github.com/jacobpatterson1549/selene-bananas/game/message"
)

type mockSocketRunner func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, inSM <-chan message.Socket) <-chan message.Message

func (m mockSocketRunner) Run(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message, inSM <-chan message.Socket) <-chan message.Message {
	return m(ctx, wg, in, inSM)
}

type mockGameRunner func(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message

func (m mockGameRunner) Run(ctx context.Context, wg *sync.WaitGroup, in <-chan message.Message) <-chan message.Message {
	return m(ctx, wg, in)
}
