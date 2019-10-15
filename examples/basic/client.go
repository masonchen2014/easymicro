package main

import (
	"context"
	"time"

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
	// #1
	client, err := client.NewClient("tcp", ":8972", "Arith")
	if err != nil {
		log.Panic(err)
		return
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
	client.Call(context.Background(), "Mul", args, reply)
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)
	client.Close()

	time.Sleep(20 * time.Second)
}
