package main

import (
	"google.golang.org/grpc"

	"inventory_srv/global"
	_ "inventory_srv/global"
	"inventory_srv/logic"
	"inventory_srv/proto"
)

func main() {
	g := grpc.NewServer()
	s := &logic.InventoryServer{}
	proto.RegisterInventoryServer(g, s)

	global.InitRPCServer(g)
}
