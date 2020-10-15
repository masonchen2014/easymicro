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
	client, err := client.NewClient(&client.ClientConfig{
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

	// #5
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	log.Infof("client call at time %d", time.Now().Unix())
	err = client.Call(ctx, "Mul", args, reply)
	if err != nil {
		//log.Fatalf("failed to call: %v", err)
		log.Error(err)
	}

	log.Infof("%d * %d = %d", args.A, args.B, reply.C)

	time.Sleep(2 * time.Second)

	args = &Args{
		A: 25,
		B: 15,
	}
	for i := 1; i <= 50; i++ {
		client.Call(context.Background(), "Mul", args, reply)
		log.Infof("%d * %d = %d", args.A, args.B, reply.C)
	}

	time.Sleep(30 * time.Second)
}
