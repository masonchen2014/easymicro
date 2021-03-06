package main

import (
	"context"
	"time"

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
	cli, err := client.NewDiscoveryClient(
		&client.DiscoverClientConfig{
			Endpoints: []string{
				"http://127.0.0.1:22379",
			},
			ServiceName: "Arith",
		})

	if err != nil {
		panic(err)
	}
	time.Sleep(10 * time.Second)

	for i := 0; i < 100; i++ {
		args := &Args{
			A: 25,
			B: 15,
		}
		reply := &Reply{}
		cli.Call(context.Background(), "Mul", args, reply)
		log.Infof("%d * %d = %d", args.A, args.B, reply.C)
		time.Sleep(3 * time.Second)
	}

	time.Sleep(2000 * time.Second)
}
