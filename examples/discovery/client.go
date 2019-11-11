package main

import (
	"context"
	"time"

	dis "github.com/masonchen2014/easymicro/discovery"
	"github.com/masonchen2014/easymicro/log"

	"github.com/masonchen2014/easymicro/client"
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

	//call 1
	cli.Call(context.Background(), "Mul", args, reply)
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	args = &Args{
		A: 35,
		B: 25,
	}
	time.Sleep(3 * time.Second)
	//call 2
	cli.Call(context.Background(), "Mul", args, reply)
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	args = &Args{
		A: 35,
		B: 35,
	}
	time.Sleep(3 * time.Second)

	//call 3
	cli.Call(context.Background(), "Mul", args, reply)
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	args = &Args{
		A: 35,
		B: 45,
	}
	time.Sleep(3 * time.Second)

	//call 4
	cli.Call(context.Background(), "Mul", args, reply)
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)
	time.Sleep(1 * time.Second)

	time.Sleep(2000 * time.Second)
}
