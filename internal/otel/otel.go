package otel

import "context"

type Span struct{}

func StartSpan(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, Span{}
}

func (Span) End() {}

func (Span) SetAttribute(_ string, _ any) {}
