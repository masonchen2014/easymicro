package metadata

import (
	"context"
	"strings"
)

type mdClientKey struct{}
type mdServersKey struct{}
type replyMessageKey struct{}

func New(m map[string]string) map[string]string {
	md := map[string]string{}
	for k, val := range m {
		key := strings.ToLower(k)
		md[key] = val
	}
	return md
}

// NewClientMdContext creates a new context with client md attached.
func NewClientMdContext(ctx context.Context, md map[string]string) context.Context {
	return context.WithValue(ctx, mdClientKey{}, md)
}

// FromClientMdContext get  md from client context.
func FromClientMdContext(ctx context.Context) (md map[string]string, ok bool) {
	md, ok = ctx.Value(mdClientKey{}).(map[string]string)
	return
}

// NewServerMdContext get  md from client context.
func NewServerMdContext(ctx context.Context, md map[string]string) context.Context {
	return context.WithValue(ctx, mdServersKey{}, md)
}

// FromServerMdContext get incoming md from server context.
func FromServerMdContext(ctx context.Context) (md map[string]string, ok bool) {
	md, ok = ctx.Value(mdServersKey{}).(map[string]string)
	return
}
