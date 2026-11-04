// Package trace describes request correlation, not a working telemetry pipeline.
package trace

import "context"

type Event struct {
	RequestID  string
	TraceID    string
	ChannelID  string
	Model      string
	Attempt    int
	StatusCode int
	DurationMS int64
	ErrorCode  string
}

// Recorder deliberately excludes credentials and request/response bodies.
type Recorder interface {
	Record(context.Context, Event) error
}
