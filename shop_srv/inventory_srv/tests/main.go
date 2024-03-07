package main

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/grpc"

	"inventory_srv/proto"
)

var invClient proto.InventoryClient
var conn *grpc.ClientConn

func TestSell(wg *sync.WaitGroup) {
	defer wg.Done()
	_, err := invClient.Sell(context.Background(), &proto.SellInfo{
		GoodsInfo: []*proto.GoodsInvInfo{
			{GoodsId: 1, Num: 1},
			//{GoodsId: 422, Num: 30},
		},
	})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("库存扣减成功")
}

func init() {
	var err error
	conn, err = grpc.Dial("10.2.178.13:50053", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	invClient = proto.NewInventoryClient(conn)
}

func main() {

	//并发情况之下 库存无法正确的扣减
	var wg sync.WaitGroup
	wg.Add(30)
	for i := 0; i < 30; i++ {
		go TestSell(&wg)
	}

	//TestInvDetail(421)
	//TestReback()

	wg.Wait()

	conn.Close()
}
