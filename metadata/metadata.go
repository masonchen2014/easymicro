package metadata

import (
	"context"
	"strconv"
	"strings"

	"github.com/masonchen2014/easymicro/share"
	"github.com/opentracing/basictracer-go"
	"github.com/opentracing/opentracing-go"
)

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
	return context.WithValue(ctx, share.ReqMetaDataKey{}, md)
} 

// FromClientMdContext get  md from client context.
func FromClientMdContext(ctx context.Context) (md map[string]string, ok bool) {
	md, ok = ctx.Value(share.ReqMetaDataKey{}).(map[string]string)
	return
}

// NewServerMdContext get  md from client context.
/*func NewServerMdContext(ctx context.Context, md map[string]string) context.Context {
	return context.WithValue(ctx, share.ResMetaDataKey{}, md)
}

// FromServerMdContext get incoming md from server context.
func FromServerMdContext(ctx context.Context) (md map[string]string, ok bool) {
	md, ok = ctx.Value(share.ResMetaDataKey{}).(map[string]string)
	return
}*/

func NewClientSpanContext(ctx context.Context, span opentracing.Span) (context.Context, error) {
	sc, ok := span.Context().(basictracer.SpanContext)
	if !ok {
		return nil, opentracing.ErrInvalidSpanContext
	}
	md := map[string]string{
		share.FieldNameTraceID: strconv.FormatUint(sc.TraceID, 16),
		share.FieldNameSpanID:  strconv.FormatUint(sc.SpanID, 16),
		share.FieldNameSampled: strconv.FormatBool(sc.Sampled),
	}

	for k, v := range sc.Baggage {
		md[share.PrefixBaggage+k] = v
	}
	return context.WithValue(ctx, share.SpanMetaDataKey{}, md), nil
}

// FromClientSpanContext get  SpanContext from client context.
func FromClientSpanContext(ctx context.Context) (opentracing.SpanContext, error) {
	sc, ok := ctx.Value(share.SpanMetaDataKey{}).(basictracer.SpanContext)
	if !ok {
		return nil, opentracing.ErrInvalidSpanContext
	}

	return sc, nil
}
