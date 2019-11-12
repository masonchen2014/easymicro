package main

import (
	"context"
	"math/rand"
	"net/http"
	"time"

	"github.com/masonchen2014/easymicro/metrics"
	"github.com/masonchen2014/easymicro/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Args struct {
	A int
	B int
}

type Reply struct {
	C int
}

type Arith int

var (
	fieldKeys    = []string{"method"}
	requestCount = metrics.NewCounterFrom(prometheus.CounterOpts{
		Namespace: "easy_micro",
		Subsystem: "arith",
		Name:      "request_count",
		Help:      "Number of requests received.",
	}, fieldKeys)
	requestLatency = metrics.NewHistogramFrom(prometheus.HistogramOpts{
		Namespace: "easy_micro",
		Subsystem: "arith",
		Name:      "request_latency_microseconds",
		Help:      "Total duration of requests in microseconds.",
		Buckets:   []float64{1, 2, 3, 5, 10},
	}, fieldKeys)
)

func (t *Arith) Mul(ctx context.Context, args *Args, reply *Reply) error {
	defer func(begin time.Time) {
		lvs := []string{"method", "mul"}
		requestCount.With(lvs...).Add(1)
		requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	rand.Seed(time.Now().Unix())
	s := rand.Intn(3)
	time.Sleep(time.Duration(s) * time.Second)
	reply.C = args.A * args.B
	return nil
}

func main() {
	s := server.NewServer()
	s.RegisterName("Arith", new(Arith))
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":8972", nil)

	s.Serve("tcp", ":8973")
	//	s.Register(new(Arith), "")

}
