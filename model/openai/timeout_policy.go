package openai

import (
	"context"
	"time"
)

const minimumTimeoutSeconds = 1

type TimeoutPolicy struct {
	TimeoutSeconds int
}

func NewTimeoutPolicy(timeoutSeconds int) TimeoutPolicy {
	if timeoutSeconds < minimumTimeoutSeconds {
		timeoutSeconds = DefaultTimeoutSeconds
	}
	return TimeoutPolicy{TimeoutSeconds: timeoutSeconds}
}

func (p TimeoutPolicy) Duration() time.Duration {
	return time.Duration(p.TimeoutSeconds) * time.Second
}

func (p TimeoutPolicy) WithTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, p.Duration())
}

func (p TimeoutPolicy) HasDeadline(parent context.Context) bool {
	if parent == nil {
		return false
	}
	_, ok := parent.Deadline()
	return ok
}
