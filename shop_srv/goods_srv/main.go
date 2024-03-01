package main

import (
	"google.golang.org/grpc"

	"goods_srv/global"
	_ "goods_srv/global"
	"goods_srv/logic"
	"goods_srv/proto"
)

func main() {
	g := grpc.NewServer()
	s := &logic.GoodsServer{}
	proto.RegisterGoodsServer(g, s)

	global.InitRPCServer(g)
}
