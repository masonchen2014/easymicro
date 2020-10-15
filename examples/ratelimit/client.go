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
	cli, err := client.NewClient(&client.ClientConfig{
		Network:     "tcp",
		Address:     ":8972",
		ServicePath: "Arith",
	}, client.SetRateLimiter(&client.LimiterConfig{
		FillInterval: time.Second,
		Capacity:     5,
		Quantum:      1,
	}))

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

	for i := 0; i < 20; i++ {
		ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
		err = cli.Call(ctx, "Mul", args, reply)
		if err != nil {
			log.Errorf("call error %+v", err)
		} else {
			log.Infof("%d * %d = %d", args.A, args.B, reply.C)
		}
	}

	time.Sleep(2000 * time.Second)
}
