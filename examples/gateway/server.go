package main

import (
	"context"

	"github.com/masonchen2014/easymicro/server"
)

type Args struct {
	A int `json:"a"`
	B int `json:"b"`
}

type Reply struct {
	C int `json:"c"`
}

type Arith int

func (t *Arith) Mul(ctx context.Context, args *Args, reply *Reply) error {
	reply.C = args.A * args.B
	server.SendMetaData(ctx, map[string]string{
		"server":  "hi this is server md",
		"server2": "enjoy yourself here",
	})
	return nil
}

func main() {
	s := server.NewServer(server.SetGateWayMode())
	s.RegisterName("Arith", new(Arith))
	s.Serve("tcp", ":8972")
	//	s.Register(new(Arith), "")

}
