package main

import (
	"context"

	pb "github.com/masonchen2014/easymicro/examples/serialize/pb"
	"github.com/masonchen2014/easymicro/server"
)

type Arith struct {
}

func (t *Arith) Mul(ctx context.Context, args *pb.MulReq, reply *pb.MulReply) error {
	reply.C = args.A * args.B
	return nil
}

func main() {
	s := server.NewServer()
	s.RegisterName("Arith", new(Arith))
	s.Serve("tcp", ":8972")
	//	s.Register(new(Arith), "")

}
