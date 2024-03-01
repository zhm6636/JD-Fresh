package main

import (
	"order_srv/logic"

	"google.golang.org/grpc"

	"order_srv/global"
	_ "order_srv/global"
	"order_srv/proto"
)

func main() {
	g := grpc.NewServer()
	s := &logic.OrderServer{}
	proto.RegisterOrderServer(g, s)

	global.InitRPCServer(g)
}
