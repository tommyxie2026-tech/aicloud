package workflow

import "context"

type Engine interface {
	Start(context.Context, string) error
}
type NoopEngine struct{}

func (NoopEngine) Start(context.Context, string) error { return nil }
