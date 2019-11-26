package main

import (
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	benchmark "github.com/masonchen2014/easymicro/benchmark/grpc/pb"
	"github.com/masonchen2014/easymicro/server"
	"golang.org/x/net/context"
)

type Hello struct{}

func (t *Hello) Say(ctx context.Context, args *benchmark.BenchmarkMessage, reply *benchmark.BenchmarkMessage) (err error) {
	s := "OK"
	var i int32 = 100
	reply.Field1 = s
	reply.Field2 = i
	if *delay > 0 {
		time.Sleep(*delay)
	} else {
		runtime.Gosched()
	}
	return nil
}

var (
	host      = flag.String("s", "127.0.0.1:8973", "listened ip and port")
	delay     = flag.Duration("delay", 0, "delay to mock business processing")
	debugAddr = flag.String("d", "127.0.0.1:9982", "server ip and port")
)

func main() {
	flag.Parse()

	go func() {
		log.Println(http.ListenAndServe(*debugAddr, nil))
	}()

	s := server.NewServer()
	s.RegisterName("Hello", new(Hello))
	s.Serve("tcp", ":8973")
}
