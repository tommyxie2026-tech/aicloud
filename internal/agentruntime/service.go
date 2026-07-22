package agentruntime

import "context"

type Runtime interface {
	Run(context.Context, string) error
}
type SkeletonRuntime struct{}

func (SkeletonRuntime) Run(context.Context, string) error { return nil }
