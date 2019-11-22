package main

import "github.com/masonchen2014/easymicro/gateway"

func main() {
	gw := gateway.NewGateway(":8887", []string{"http://127.0.0.1:22379"})
	gw.StartGateway()
}
