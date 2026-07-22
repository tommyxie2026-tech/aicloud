package toolgateway

import (
	"context"
	"errors"
)

type Gateway interface {
	Invoke(context.Context, string, string) (string, error)
}
type DenyByDefault struct{}

func (DenyByDefault) Invoke(context.Context, string, string) (string, error) {
	return "", errors.New("tool execution is disabled in the skeleton runtime")
}
