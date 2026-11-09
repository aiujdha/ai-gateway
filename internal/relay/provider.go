// Package relay defines model adapters, including the local text provider.
package relay

import "context"

type Request struct {
	Model  string
	Prompt string
	Stream bool
}

type Response struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

type StreamEvent struct {
	Delta string
	Done  bool
}

// Provider represents ordinary and streaming model calls.
type Provider interface {
	Name() string
	Generate(context.Context, Request) (Response, error)
	Stream(context.Context, Request) (<-chan StreamEvent, error)
}
