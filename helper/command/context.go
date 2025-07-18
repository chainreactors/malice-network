package command

import "context"

type contextKey string

const RawInputKey contextKey = "rawInput"

func WithRawInput(ctx context.Context, input string) context.Context {
	return context.WithValue(ctx, RawInputKey, input)
}

func GetRawInput(ctx context.Context) string {
	val := ctx.Value(RawInputKey)
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}
