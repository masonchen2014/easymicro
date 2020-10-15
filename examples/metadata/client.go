package main

import (
	"context"
	"time"

	"github.com/masonchen2014/easymicro/client"
	"github.com/masonchen2014/easymicro/log"
	"github.com/masonchen2014/easymicro/metadata"

	easyclient "github.com/masonchen2014/easymicro/client"
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
	client, err := easyclient.NewClient(&client.ClientConfig{
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
	ctx := metadata.NewClientMdContext(context.Background(), map[string]string{
		"testmd":  "md",
		"testMd2": "md2",
	})

	mdFromServer := map[string]string{}
	err = client.Call(ctx, "Mul", args, reply, easyclient.GetMetadataFromServer(&mdFromServer))
	if err != nil {
		log.Error(err)
	}
	log.Infof("%d * %d = %d", args.A, args.B, reply.C)
	log.Infof("md from server %+v", mdFromServer)

	time.Sleep(2 * time.Second)
}
