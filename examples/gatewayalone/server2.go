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

func (t *Arith) Plus(ctx context.Context, args *Args, reply *Reply) error {
	server.SendMetaData(ctx, map[string]string{
		"server":  "hi this is server md",
		"server2": "context",
	})
	reply.C = args.A + args.B
	return nil
}

func main() {
	s := server.NewServer(server.SetEtcdDiscovery([]string{
		"http://127.0.0.1:22379",
		//"http://172.16.164.179:2379",
	}, "127.0.0.1:8974"))

	s.RegisterName("Arith2", new(Arith))
	s.Serve("tcp", ":8974")
	//	s.Register(new(Arith), "")

}
