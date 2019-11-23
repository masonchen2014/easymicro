package main

import (
	"context"
	"time"

	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/protocol"

	"github.com/masonchen2014/easymicro/client"
	pb "github.com/masonchen2014/easymicro/examples/serialize/pb"
)

func main() {
	// #1
	c, err := client.NewClient("tcp", ":8972", "Arith")
	if err != nil {
		log.Panic(err)
		return
	}
	defer c.Close()

	// #3
	args := &pb.MulReq{
		A: 10,
		B: 20,
	}

	// #4
	reply := &pb.MulReply{}

	c.Call(context.Background(), "Mul", args, reply, client.SetCallSerializeType(protocol.ProtoBuffer))
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	time.Sleep(20 * time.Second)
}
