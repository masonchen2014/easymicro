package main

import (
	"context"
	"time"

	dis "github.com/masonchen2014/easymicro/discovery"
	"github.com/masonchen2014/easymicro/log"
	"github.com/juju/ratelimit"

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
	}, "services/Arith/"), client.SetRateLimiter(ratelimit.NewBucketWithQuantum(1*time.Second, 5, 1)))

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
