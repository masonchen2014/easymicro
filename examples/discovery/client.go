package main

import (
	"context"
	"time"

	"github.com/easymicro/discovery"
	"github.com/easymicro/log"

	"github.com/easymicro/client"
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
	master, err := discovery.NewEtcdDiscoveryMaster([]string{
		"http://127.0.0.1:22379",
	}, "services/Arith/")
	if err != nil {
		log.Panic(err)
	}

	addr := ""
	for k, v := range master.Nodes {
		log.Infof("key %v value %v", k, v)
		addr = v.Info.Addr
	}
	log.Infof("addr is %s", addr)
	// #1
	client, err := client.NewClient("tcp", addr)
	if err != nil {
		log.Panic(err)
		return
	}
	defer client.Close()

	// #3
	args := &Args{
		A: 10,
		B: 20,
	}

	// #4
	reply := &Reply{}

	args = &Args{
		A: 25,
		B: 15,
	}
	client.Call(context.Background(), "Arith.Mul", args, reply)
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	time.Sleep(20 * time.Second)
}
