package metadata

import (
	"context"
	"strings"
)

type mdClientKey struct{}
type mdServersKey struct{}

func New(m map[string]string) map[string]string {
	md := map[string]string{}
	for k, val := range m {
		key := strings.ToLower(k)
		md[key] = val
	}
	return md
}

// NewIncomingContext creates a new context with incoming md attached.
func NewClientMdContext(ctx context.Context, md map[string]string) context.Context {
	return context.WithValue(ctx, mdClientKey{}, md)
}

// FromIncomingContext get incoming md from incoming context.
func FromClientMdContext(ctx context.Context) (md map[string]string, ok bool) {
	md, ok = ctx.Value(mdClientKey{}).(map[string]string)
	return
}

// NewIncomingContext creates a new context with incoming md attached.
func NewServerMdContext(ctx context.Context, md map[string]string) context.Context {
	return context.WithValue(ctx, mdServersKey{}, md)
}

// FromIncomingContext get incoming md from incoming context.
func FromServerMdContext(ctx context.Context) (md map[string]string, ok bool) {
	md, ok = ctx.Value(mdServersKey{}).(map[string]string)
	return
}
