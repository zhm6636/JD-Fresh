package logic

import (
	"context"
	"errors"
	"fmt"
	"time"

	"inventory_srv/global"
	"inventory_srv/model"
	"inventory_srv/proto"
)

type InventoryServer struct {
	proto.UnimplementedInventoryServer
}

func (i InventoryServer) SetInv(ctx context.Context, info *proto.GoodsInvInfo) (*proto.Empty, error) {
	var inventory = model.Inventory{}
	global.MysqlConf.DB.Where("goods = ?", info.GoodsId).First(&inventory)
	inventory.Goods = info.GoodsId
	inventory.Stocks = info.Num
	res := global.MysqlConf.DB.Save(&inventory)
	if res.Error != nil {
		return nil, res.Error

	}
	return &proto.Empty{}, nil
}

func (i InventoryServer) InvDetail(ctx context.Context, info *proto.GoodsInvInfo) (*proto.GoodsInvInfo, error) {
	var inventory = model.Inventory{}
	global.MysqlConf.DB.Where("goods = ?", info.GoodsId).First(&inventory)
	return &proto.GoodsInvInfo{
		GoodsId: inventory.Goods,
		Num:     inventory.Stocks,
	}, nil
}

func (i InventoryServer) Sell(ctx context.Context, info *proto.SellInfo) (*proto.Empty, error) {
	//for _, v := range info.GoodsInfo {
	//	var inventory = model.Inventory{}
	//	global.MysqlConf.DB.Where("goods = ?", v.GoodsId).First(&inventory)
	//	if inventory.Stocks < v.Num {
	//		return nil, errors.New("库存不足")
	//	}
	//	inventory.Stocks -= v.Num
	//	global.MysqlConf.DB.Save(&inventory)
	//}
	//return &proto.Empty{}, nil
	// 创建 Redis 客户端

	redisClient := global.RedisConf.DB

	for _, v := range info.GoodsInfo {
		// 使用 Redis 分布式锁
		lockKey := fmt.Sprintf("lock:%d", v.GoodsId)

		// 尝试获取锁，设置过期时间为 10 秒
		lock, err := redisClient.SetNX(ctx, lockKey, "locked", 10*time.Second).Result()
		if err != nil {
			return nil, err
		}

		if !lock {
			// 未获取到锁，可能有其他操作正在进行
			return nil, errors.New("无法获取锁，可能有其他操作正在进行")
		}
		defer redisClient.Del(ctx, lockKey) // 确保在函数结束时释放锁

		var inventory = model.Inventory{}
		// 注意，这里使用了Find而不是Where+First，以避免在无记录时抛出gorm.ErrRecordNotFound
		global.MysqlConf.DB.Find(&inventory, "goods = ?", v.GoodsId)

		if inventory.Stocks < v.Num {
			return nil, errors.New("库存不足")
		}

		// 扣减库存
		inventory.Stocks -= v.Num
		global.MysqlConf.DB.Save(&inventory)
	}

	return &proto.Empty{}, nil
}

func (i InventoryServer) Reback(ctx context.Context, info *proto.SellInfo) (*proto.Empty, error) {
	for _, v := range info.GoodsInfo {
		var inventory = model.Inventory{}
		global.MysqlConf.DB.Where("goods = ?", v.GoodsId).First(&inventory)
		inventory.Stocks += v.Num
		global.MysqlConf.DB.Save(&inventory)
	}
	return &proto.Empty{}, nil
}
