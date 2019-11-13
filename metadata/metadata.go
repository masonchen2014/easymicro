package metadata

import (
	"context"
	"strconv"
	"strings"

	"github.com/opentracing/basictracer-go"
	"github.com/opentracing/opentracing-go"
)

const (
	prefixTracerState = "ot-tracer-"
	prefixBaggage     = "ot-baggage-"
	prefixCommonMd    = "md-common-"

	tracerStateFieldCount = 3
	fieldNameTraceID      = prefixTracerState + "traceid"
	fieldNameSpanID       = prefixTracerState + "spanid"
	fieldNameSampled      = prefixTracerState + "sampled"
)

type mdClientKey struct{}
type mdServersKey struct{}
type replyMessageKey struct{}
type spanMdKey struct{}

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
	/*	comMd := make(map[string]string)
		for k, v := range md {
			comMd[prefixCommonMd+k] = v
		}*/
	return context.WithValue(ctx, mdClientKey{}, md)
}

func ExtractClientMdContexFromMd(ctx context.Context, md map[string]string) (context.Context, error) {
	var traceId, spanId uint64
	var sampled, hasSpanCtx bool
	var err error
	decodedBaggage := make(map[string]string)
	commonMd := make(map[string]string)
	for k, v := range md {
		switch strings.ToLower(k) {
		case fieldNameTraceID:
			hasSpanCtx = true
			traceId, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return nil, opentracing.ErrSpanContextCorrupted
			}
		case fieldNameSpanID:
			hasSpanCtx = true
			spanId, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return nil, opentracing.ErrSpanContextCorrupted
			}
		case fieldNameSampled:
			hasSpanCtx = true
			sampled, err = strconv.ParseBool(v)
			if err != nil {
				return nil, opentracing.ErrSpanContextCorrupted
			}
		default:
			lowercaseK := strings.ToLower(k)
			if strings.HasPrefix(lowercaseK, prefixBaggage) {
				hasSpanCtx = true
				decodedBaggage[strings.TrimPrefix(lowercaseK, prefixBaggage)] = v
			} else {
				commonMd[k] = v
			}
		}

	}
	if len(commonMd) > 0 {
		ctx = context.WithValue(ctx, mdClientKey{}, md)
	}

	if hasSpanCtx {
		spanCtx := basictracer.SpanContext{
			TraceID: traceId,
			SpanID:  spanId,
			Sampled: sampled,
			Baggage: decodedBaggage,
		}
		ctx = context.WithValue(ctx, spanMdKey{}, spanCtx)
	}

	return ctx, nil
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

func NewClientContextWithSpan(ctx context.Context, span opentracing.Span) (context.Context, error) {
	sc, ok := span.Context().(basictracer.SpanContext)
	if !ok {
		return nil, opentracing.ErrInvalidSpanContext
	}
	md := map[string]string{
		fieldNameTraceID: strconv.FormatUint(sc.TraceID, 16),
		fieldNameSpanID:  strconv.FormatUint(sc.SpanID, 16),
		fieldNameSampled: strconv.FormatBool(sc.Sampled),
	}

	for k, v := range sc.Baggage {
		md[prefixBaggage+k] = v
	}
	return context.WithValue(ctx, spanMdKey{}, md), nil
}

// FromClientSpanContext get  md from client context.
func FromClientSpanContext(ctx context.Context) (opentracing.SpanContext, error) {
	sc, ok := ctx.Value(spanMdKey{}).(basictracer.SpanContext)
	if !ok {
		return nil, opentracing.ErrInvalidSpanContext
	}

	return sc, nil
}

func ExtractMdFromClientCtx(ctx context.Context) map[string]string {
	md := make(map[string]string)
	comMd, ok := ctx.Value(mdClientKey{}).(map[string]string)
	if ok {
		for k, v := range comMd {
			md[k] = v
		}
	}
	spanMd, ok := ctx.Value(spanMdKey{}).(map[string]string)
	if ok {
		for k, v := range spanMd {
			md[k] = v
		}
	}
	return md
}
