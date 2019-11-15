package main

import (
	"context"
	"time"

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
	time.Sleep(5 * time.Second)
	return nil
}

func (t *Arith) Add(ctx context.Context, args *Args, reply *Reply) error {
	reply.C = args.A + args.B
	return nil
}

func main() {
	s := server.NewServer(server.SetEtcdDiscovery([]string{
		"http://127.0.0.1:22379",
		//"http://172.16.164.179:2379",
	}), "127.0.0.1:8972")

	s.RegisterName("Arith", new(Arith))
	s.Serve("tcp", ":8972")
	//	s.Register(new(Arith), "")

}
