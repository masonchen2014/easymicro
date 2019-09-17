package main

import (
	"context"
	"time"

	"github.com/easymicro/log"
	"github.com/easymicro/metadata"

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
	ctx := metadata.NewMdContext(context.Background(), map[string]string{
		"testmd":  "md",
		"testMd2": "md2",
	})
	err = client.Call(ctx, "Arith.Mul", args, reply)
	if err != nil {
		log.Error(err)
	}
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)
	time.Sleep(2 * time.Second)
}
