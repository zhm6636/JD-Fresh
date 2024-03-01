package main

import (
	"google.golang.org/grpc"

	"user_srv/global"
	_ "user_srv/global"
	"user_srv/logic"
	"user_srv/proto"
)

func main() {
	g := grpc.NewServer()
	s := &logic.UserServer{}
	proto.RegisterUserServer(g, s)
	global.InitRPCServer(g)
}
