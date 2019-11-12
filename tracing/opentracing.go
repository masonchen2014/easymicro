package tracing

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

	tracerStateFieldCount = 3
	fieldNameTraceID      = prefixTracerState + "traceid"
	fieldNameSpanID       = prefixTracerState + "spanid"
	fieldNameSampled      = prefixTracerState + "sampled"
)

type spanClientKey struct{}

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
	return context.WithValue(ctx, spanClientKey{}, md), nil
}

// FromClientMdContext get  md from client context.
func FromClientSpanContext(ctx context.Context) (opentracing.SpanContext, error) {
	md, ok := ctx.Value(spanClientKey{}).(map[string]string)
	if !ok {
		return nil, opentracing.ErrInvalidSpanContext
	}

	var traceId, spanId uint64
	var sampled bool
	var err error
	decodedBaggage := make(map[string]string)
	for k, v := range md {
		switch strings.ToLower(k) {
		case fieldNameTraceID:
			traceId, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return nil, opentracing.ErrSpanContextCorrupted
			}
		case fieldNameSpanID:
			spanId, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return nil, opentracing.ErrSpanContextCorrupted
			}
		case fieldNameSampled:
			sampled, err = strconv.ParseBool(v)
			if err != nil {
				return nil, opentracing.ErrSpanContextCorrupted
			}
		default:
			lowercaseK := strings.ToLower(k)
			if strings.HasPrefix(lowercaseK, prefixBaggage) {
				decodedBaggage[strings.TrimPrefix(lowercaseK, prefixBaggage)] = v
			}
		}

	}
	return basictracer.SpanContext{
		TraceID: traceId,
		SpanID:  spanId,
		Sampled: sampled,
		Baggage: decodedBaggage,
	}, nil
}
