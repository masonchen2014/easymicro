package main

import (
	"context"
	"math/rand"
	"time"

	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/metadata"
	"github.com/masonchen2014/easymicro/server"
	"github.com/opentracing/opentracing-go"
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

func (t *Arith) Add(ctx context.Context, args *Args, reply *Reply) error {
	var sp opentracing.Span
	sc, err := metadata.FromClientSpanContext(ctx)
	log.Infof("add span ctx %+v", sc)
	if err != nil {
		sp = opentracing.StartSpan("/Add")
	} else {
		sp = opentracing.StartSpan("/Add", opentracing.ChildOf(sc))
	}
	defer sp.Finish()
	reply.C = args.A + args.B
	randNum := rand.Intn(5)
	time.Sleep(time.Duration(randNum) * time.Second)
	return nil
}

func main() {
	tracer := appdashot.NewTracer(appdash.NewRemoteCollector(":3309"))
	opentracing.InitGlobalTracer(tracer)
	s := server.NewServer()
	s.RegisterName("Arith", new(Arith))
	s.Serve("tcp", ":9982")
	//	s.Register(new(Arith), "")

}
