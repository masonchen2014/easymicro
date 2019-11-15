package main

import (
	"context"

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
	reply.C = args.A * args.B
	return nil
}

func main() {
	s := server.NewServer()
	s.RegisterName("Arith", new(Arith))
	s.Serve("tcp", ":8973")
	//	s.Register(new(Arith), "")

}
