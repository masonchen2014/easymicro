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
	client, err := client.NewClient("tcp", ":8972")
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

	// #5
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = client.Call(ctx, "Arith.Mul", args, reply)
	if err != nil {
		//log.Fatalf("failed to call: %v", err)
		log.Error(err)
	}
	defer func() {
		cancel()
	}()

	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	time.Sleep(2 * time.Second)

	args = &Args{
		A: 25,
		B: 15,
	}
	client.Call(context.Background(), "Arith.Mul", args, reply)
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	time.Sleep(20 * time.Second)
}