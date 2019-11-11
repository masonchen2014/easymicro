package main

import (
	"context"

	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/metadata"
	"github.com/masonchen2014/easymicro/server"
)

type Args struct {
	A int
	B int
}

type Reply struct {
	C int
}

type Arith int

func (t *Arith) Mul(ctx context.Context, args *Args, reply *Reply) error {
	md, b := metadata.FromClientMdContext(ctx)
	if b {
		log.Infof("get md %+v from context", md)
	}
	reply.C = args.A * args.B
	server.SendMetaData(ctx, map[string]string{
		"server": "hi this is server md",
	})
	return nil
}

func main() {
	s := server.NewServer()
	s.RegisterName("Arith", new(Arith))
	s.Serve("tcp", ":8972")
	//	s.Register(new(Arith), "")

}
