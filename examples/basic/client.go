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
	// #1
	client, err := client.NewClient(
		&client.ClientConfig{
			Network:     "tcp",
			Address:     ":8972",
			ServicePath: "Arith",
		})
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
	client.Call(context.Background(), "Mul", args, reply)
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	time.Sleep(200 * time.Second)
}
