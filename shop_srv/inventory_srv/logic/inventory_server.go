package logic

import (
	"context"
	"fmt"
	"log"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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
	//redisClient := global.RedisConf.DB
	//for _, v := range info.GoodsInfo {
	//	lockKey := fmt.Sprintf("lock:%d", v.GoodsId)
	//
	//	// 尝试获取锁，设置过期时间为 10 秒
	//	lock, err := redisClient.SetNX(ctx, lockKey, "locked", 10*time.Second).Result()
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	if !lock {
	//		// 未获取到锁，将 lockKey 的值设置为列表，然后阻塞等待直到获取到锁
	//		//_, err = redisClient.LPush(ctx, lockKey, "blocked").Result()
	//		//if err != nil {
	//		//	return nil, err
	//		//}
	//		//
	//		//_, err = redisClient.BLPop(ctx, 0, lockKey).Result()
	//		//if err != nil {
	//		//	return nil, err
	//		//}
	//		//
	//		//// 获取到锁后，将列表中的元素出栈
	//		//_, _ = redisClient.RPop(ctx, lockKey).Result()
	//		return nil, errors.New("未获取到锁")
	//	}
	//	defer redisClient.Del(ctx, lockKey) // 确保在函数结束时释放锁

	//redisClient := global.RedisConf.DB
	//redisClient.Set(context.Background(), "123456", 111, 0)
	//for _, v := range info.GoodsInfo {
	//	lockKey := fmt.Sprintf("lock:%d", v.GoodsId)
	//
	//	// 使用 WATCH 监视锁的变化
	//	watchRes := redisClient.Watch(ctx, func(tx *redis.Tx) error {
	//		// 检查锁是否已经被获取
	//		currentLockValue, err := tx.Get(ctx, lockKey).Result()
	//		if err != nil && err != redis.Nil {
	//			return err
	//		}
	//
	//		if currentLockValue == "" {
	//			// 锁未被获取，可以尝试上锁
	//			_, err := tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
	//				// 上锁并设置过期时间为 10 秒
	//				pipe.Set(ctx, lockKey, "locked", 10*time.Second)
	//				return nil
	//			})
	//
	//			return err
	//		}
	//
	//		// 锁已经被获取，放弃事务
	//		return redis.TxFailedErr
	//	}, lockKey)
	//
	//	if watchRes == redis.TxFailedErr {
	//		// 事务失败，说明锁已经被其他客户端获取，阻塞等待
	//		_, err := redisClient.BLPop(ctx, 0, lockKey).Result()
	//		if err != nil {
	//			return nil, err
	//		}
	//		// 获取到锁后继续执行下面的代码
	//	} else if watchRes != nil {
	//		// WATCH 操作失败，处理错误
	//		return nil, watchRes
	//	}
	//
	//	// 释放锁的逻辑
	//	defer redisClient.Del(ctx, lockKey)
	//
	//	var inventory = model.Inventory{}
	//	// 注意，这里使用了Find而不是Where+First，以避免在无记录时抛出gorm.ErrRecordNotFound
	//	global.MysqlConf.DB.Find(&inventory, "goods = ?", v.GoodsId)
	//
	//	if inventory.Stocks < v.Num {
	//		return nil, errors.New("库存不足")
	//	}
	//
	//	// 扣减库存
	//	inventory.Stocks -= v.Num
	//	global.MysqlConf.DB.Save(&inventory)
	//}
	//
	//return &proto.Empty{}, nil

	//tx := global.MysqlConf.DB.Begin()
	//for _, goodInfo := range info.GoodsInfo {
	//	var inv model.Inventory
	//	if result := tx.Where(&model.Inventory{Goods: goodInfo.GoodsId}).First(&inv); result.RowsAffected == 0 {
	//		tx.Rollback() //回滚之前的操作
	//		return nil, status.Errorf(codes.InvalidArgument, "没有库存信息")
	//	}
	//
	//	//判断库存是否充足
	//	if inv.Stocks < goodInfo.Num {
	//		tx.Rollback() //回滚之前的操作
	//		return nil, status.Errorf(codes.ResourceExhausted, "库存不足")
	//	}
	//	//扣减， 会出现数据不一致的问题 - 锁，分布式锁
	//	//扣减库存
	//	tx.Model(&inv).Update("stocks", inv.Stocks-goodInfo.Num)
	//	inv.Stocks -= goodInfo.Num
	//	tx.Save(&inv)
	//}
	//tx.Commit() // 需要自己手动提交操作
	//return &proto.Empty{}, nil

	client := redis.NewClient(&redis.Options{
		Addr: "42.192.108.133:6379",
	})
	pool := goredis.NewPool(client)
	rs := redsync.New(pool)

	tx := global.MysqlConf.DB.Begin()
	//定义库存扣减历史表结构体
	stockDetail := model.StockSellDetail{
		OrderSn: info.OrderSn,
		Status:  1,
	}

	var details []model.GoodsDetail

	for _, goodInfo := range info.GoodsInfo {
		var inv model.Inventory

		//处理订单购买哪些商品和几件数量
		details = append(details, model.GoodsDetail{
			Goods: goodInfo.GoodsId,
			Num:   goodInfo.Num,
		})

		// 获取分布式锁
		mutex := rs.NewMutex(fmt.Sprintf("goods_%d", goodInfo.GoodsId))
		if err := mutex.Lock(); err != nil {
			return nil, status.Errorf(codes.Internal, "获取redis分布式锁失败")
		}
		defer func() {
			// 确保锁被释放
			if _, err := mutex.Unlock(); err != nil {
				// 这里应该记录日志，因为释放锁失败可能会导致其他操作出现问题
				log.Printf("释放redis分布式锁失败: %v", err)
			}
		}()

		if result := tx.Where(&model.Inventory{Goods: goodInfo.GoodsId}).First(&inv); result.RowsAffected == 0 {
			tx.Rollback()
			return nil, status.Errorf(codes.InvalidArgument, "没有库存信息")
		}

		if inv.Stocks < goodInfo.Num {
			tx.Rollback()
			return nil, status.Errorf(codes.ResourceExhausted, "库存不足")
		}
		inv.Stocks -= goodInfo.Num //扣减本地库存
		//inv.ReduceStock += goodInfo.Num //把本地冻结库存添加上去
		tx.Save(&inv)
	}

	//在扣减库存之后我们保存历史记录表
	stockDetail.Detail = details
	tx.Create(&stockDetail)

	tx.Commit()
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
