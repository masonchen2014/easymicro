package main

import (
	"context"
	"time"

	"github.com/easymicro/log"

	easyclient "github.com/easymicro/client"
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
	client, err := easyclient.NewClient("tcp", "172.16.164.248:8972")
	if err != nil {
		log.Error(err)
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
	for i := 0; i < 5; i++ {
		log.Infof("client call at time %d", time.Now().Unix())
		err = client.Call(context.Background(), "Arith.Mul", args, reply)
		if err != nil {
			log.Error(err)
		}
		log.Infof("%d * %d = %d", args.A, args.B, reply.C)

		time.Sleep(2 * time.Second)
	}

	//time.Sleep(10 * time.Second)

	/*for i := 0; i < 5; i++ {
		log.Infof("client call at time %d", time.Now().Unix())
		err = client.Call(context.Background(), "Arith.Mul", args, reply)
		if err != nil {
			log.Error(err)
		}
		log.Infof("%d * %d = %d", args.A, args.B, reply.C)

		time.Sleep(2 * time.Second)
	}*/ //

	time.Sleep(700 * time.Second)
}
