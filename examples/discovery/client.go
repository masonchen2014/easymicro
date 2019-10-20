package main

import (
	"context"
	"time"

	dis "github.com/easymicro/discovery"
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
	cli, err := client.NewDiscoveryClient("Arith", dis.NewEtcdDiscoveryMaster([]string{
		"http://127.0.0.1:22379",
	}, "services/Arith/"))

	if err != nil {
		panic(err)
	}

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
	cli.Call(context.Background(), "Mul", args, reply)
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	time.Sleep(2000 * time.Second)
}
