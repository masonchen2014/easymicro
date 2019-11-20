package main

import (
	"context"
	"math/rand"
	"time"

	"github.com/masonchen2014/easymicro/client"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/metadata"
	"github.com/masonchen2014/easymicro/server"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"sourcegraph.com/sourcegraph/appdash"
	appdashot "sourcegraph.com/sourcegraph/appdash/opentracing"
)

type Args struct {
	A int
	B int
}

type Reply struct {
	C int
}

type Arith int

var addClient *client.Client

func init() {
	client, err := client.NewClient("tcp", ":9982", "Arith")
	if err != nil {
		log.Panic(err)
		return
	}
	addClient = client
}

func doSomething() {

}

func (t *Arith) Mul(ctx context.Context, args *Args, reply *Reply) error {
	span := opentracing.StartSpan("/Mul")
	defer span.Finish()
	reply.C = args.A * args.B
	randNum := rand.Intn(2)
	time.Sleep(time.Duration(randNum) * time.Second)
	replyAdd := &Reply{}
	ctxTm, _ := context.WithTimeout(context.Background(), 3*time.Second)
	//set span
	ec := server.FromClientConnContext(ctx)
	if ec != nil {
		ext.PeerAddress.Set(span, ec.RemoteAddr().String())
	}

	ctxTm, err := metadata.NewClientSpanContext(ctxTm, span)
	log.Infof("ctxTm with span %+v", ctxTm)
	if err != nil {
		ext.Error.Set(span, true)                                       // Tag the span as errored
		span.LogEventWithPayload("NewClientContextWithSpan error", err) // Log the error
	}
	//span.Tracer().Inject(span.Context(), opentracing.TextMap,opentracing.HTTPHeaderTextMapCarrier(asyncReq.Header)))
	if err := addClient.Call(ctxTm, "Add", args, replyAdd); err != nil {
		ext.Error.Set(span, true)                               // Tag the span as errored
		span.LogEventWithPayload("call add service error", err) // Log the error
	} else {
		log.Infof("arith add  %d and  %d equals %d", args.A, args.B, replyAdd.C)
	}
	return nil
}

func main() {
	tracer := appdashot.NewTracer(appdash.NewRemoteCollector(":3309"))
	opentracing.InitGlobalTracer(tracer)

	s := server.NewServer()
	s.RegisterName("Arith", new(Arith))
	s.Serve("tcp", ":9981")
	//	s.Register(new(Arith), "")

}
