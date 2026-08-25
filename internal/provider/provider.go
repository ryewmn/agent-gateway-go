package provider

import (
	"context"
	"errors"

	"github.com/ryewmn/agent-gateway-go/internal/model"
)

var (
	ErrUnavailable = errors.New("provider unavailable")
	ErrRateLimited = errors.New("provider rate limited")
)

// Provider is safe for concurrent use.
type Provider interface {
	Name() string
	Chat(context.Context, model.ChatRequest) (model.ChatResponse, error)
}

// Temporary marks errors that may succeed on another attempt or provider.
type Temporary interface {
	error
	Temporary() bool
}

func IsRetryable(err error) bool {
	return errors.Is(err, ErrUnavailable) || errors.Is(err, ErrRateLimited) || errors.Is(err, context.DeadlineExceeded)
}
