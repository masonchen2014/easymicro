package server

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/masonchen2014/easymicro/protocol"
	"github.com/masonchen2014/easymicro/share"
	"github.com/opentracing/basictracer-go"
	"github.com/opentracing/opentracing-go"
)

type ReplyMessageDataKey struct{}

type ServerDataKey struct{}

type ConnDataKey struct{}

func SendMetaData(ctx context.Context, md map[string]string) error {
	replyMessage, ok := ctx.Value(ReplyMessageDataKey{}).(*protocol.Message)
	if !ok {
		return errors.New("failed to fetch reply message from context")
	}

	replyMessage.Metadata = md
	return nil
}

func FromClientConnContext(ctx context.Context) (*easyConn, bool) {
	ec, ok := ctx.Value(ConnDataKey{}).(*easyConn)
	return ec, ok
}

func extractClientMdContexFromMd(ctx context.Context, md map[string]string) (context.Context, error) {
	var traceId, spanId uint64
	var sampled, hasSpanCtx bool
	var err error
	decodedBaggage := make(map[string]string)
	commonMd := make(map[string]string)
	for k, v := range md {
		switch strings.ToLower(k) {
		case share.FieldNameTraceID:
			hasSpanCtx = true
			traceId, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return nil, opentracing.ErrSpanContextCorrupted
			}
		case share.FieldNameSpanID:
			hasSpanCtx = true
			spanId, err = strconv.ParseUint(v, 16, 64)
			if err != nil {
				return nil, opentracing.ErrSpanContextCorrupted
			}
		case share.FieldNameSampled:
			hasSpanCtx = true
			sampled, err = strconv.ParseBool(v)
			if err != nil {
				return nil, opentracing.ErrSpanContextCorrupted
			}
		default:
			lowercaseK := strings.ToLower(k)
			if strings.HasPrefix(lowercaseK, share.PrefixBaggage) {
				hasSpanCtx = true
				decodedBaggage[strings.TrimPrefix(lowercaseK, share.PrefixBaggage)] = v
			} else {
				commonMd[k] = v
			}
		}

	}
	if len(commonMd) > 0 {
		ctx = context.WithValue(ctx, share.ReqMetaDataKey{}, md)
	}

	if hasSpanCtx {
		spanCtx := basictracer.SpanContext{
			TraceID: traceId,
			SpanID:  spanId,
			Sampled: sampled,
			Baggage: decodedBaggage,
		}
		ctx = context.WithValue(ctx, share.SpanMetaDataKey{}, spanCtx)
	}

	return ctx, nil
}
