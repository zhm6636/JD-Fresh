package main

import (
	"userop_srv/logic"
	"userop_srv/proto"

	"google.golang.org/grpc"

	"userop_srv/global"
	_ "userop_srv/global"
)

func main() {
	g := grpc.NewServer()
	s := &logic.UserOpServer{}
	proto.RegisterMessageServer(g, s)
	proto.RegisterUserFavServer(g, s)
	proto.RegisterAddressServer(g, s)

	global.InitRPCServer(g)
}
