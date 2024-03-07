package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"gorm.io/gorm"

	"order_srv/global"

	"order_srv/model"
	"order_srv/proto"
	goods "order_srv/proto/goods_srv"
	inventory "order_srv/proto/inventory_srv"
)

func (o OrderServer) Create(ctx context.Context, req *proto.OrderRequest) (*proto.OrderInfoResponse, error) {
	/*
	   新建订单
	       1. 从购物车中获取到选中的商品
	       2. 商品的价格自己查询 - 访问商品服务 (跨微服务)
	       3. 库存的扣减 - 访问库存服务 (跨微服务)
	       4. 订单的基本信息表 - 订单的商品信息表
	       5. 从购物车中删除已购买的记录
	*/
	//定一个切片用来保存购物车商品数据
	var shopCarts []model.ShoppingCart
	//定义一个切片存放购物车下的商品id
	var goodsIds []int32
	//定义商品数据字典
	goodsNumsMap := make(map[int32]int32)
	if result := global.MysqlConf.DB.Where(&model.ShoppingCart{User: req.UserId, Checked: true}).Find(&shopCarts); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.Internal, "没有选中任何商品")
	}
	//得到购物车的所有商品id
	for _, shopCart := range shopCarts {
		goodsIds = append(goodsIds, shopCart.Goods)
		goodsNumsMap[shopCart.Goods] = shopCart.Nums
	}
	//去商品微服务查询商品价格
	goods, err := global.GoodsClient.BatchGetGoods(context.Background(), &goods.BatchGoodsIdInfo{
		Id: goodsIds,
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "没有任何商品信息")
	}
	//订单的总金额 = 所有商品的金额加一起 (商品的价格（goods）*购物车商品的数量(shopCarts.nums))
	var orderMount float32
	//定义一个切片 保存扣减商品和数量信息
	var goodsInvInfo []*inventory.GoodsInvInfo
	//定义一个切片 保存订单购买的所有商品
	var orderGoods []*model.OrderGoods
	for _, good := range goods.Data {
		orderMount += good.ShopPrice * float32(goodsNumsMap[good.Id])
		orderGoods = append(orderGoods, &model.OrderGoods{
			Goods:      good.Id,
			GoodsName:  good.Name,
			GoodsImage: good.GoodsFrontImage,
			GoodsPrice: good.ShopPrice,
			Nums:       goodsNumsMap[good.Id],
		})
		goodsInvInfo = append(goodsInvInfo, &inventory.GoodsInvInfo{
			GoodsId: good.Id,
			Num:     goodsNumsMap[good.Id],
		})
	}
	//库存的扣减
	_, err = global.InventoryClient.Sell(context.Background(), &inventory.SellInfo{
		GoodsInfo: goodsInvInfo,
	})
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "库存扣减失败")
	}
	tx := global.MysqlConf.DB.Begin()
	//创建订单基本信息表
	OrderInfo := model.OrderInfo{
		User:         req.UserId,
		OrderSn:      GenerateOrderSn(req.UserId),
		Status:       "PAYING",
		OrderMount:   orderMount,
		Address:      req.Address,
		SignerName:   req.Name,
		SingerMobile: req.Mobile,
		Post:         req.Post,
	}
	if result := tx.Create(&OrderInfo); result.RowsAffected == 0 {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "创建订单失败")
	}
	//创建订单商品表
	for _, orderGood := range orderGoods {
		orderGood.Order = OrderInfo.ID
	}
	//多条数据，最好不要循环入库，来使用批量入库
	if result := tx.CreateInBatches(orderGoods, 100); result.RowsAffected == 0 {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "创建订单商品失败")
	}
	//从购物车中删除已购买的记录
	if result := tx.Where(&model.ShoppingCart{User: req.UserId, Checked: true}).Delete(&shopCarts); result.RowsAffected == 0 {
		tx.Rollback()
		return nil, status.Errorf(codes.Internal, "删除购物车商品失败")
	}
	//返回结果
	OrderInfoResponse := &proto.OrderInfoResponse{
		Id:      OrderInfo.ID,
		UserId:  OrderInfo.User,
		OrderSn: OrderInfo.OrderSn,
		PayType: OrderInfo.PayType,
		Status:  OrderInfo.Status,
		Post:    OrderInfo.Post,
		Total:   OrderInfo.OrderMount,
		Address: OrderInfo.Address,
		Name:    OrderInfo.SignerName,
		Mobile:  OrderInfo.SingerMobile,
		//AddTime: OrderInfo.CreatedAt.Format("2006-01-02 15:04:05"),
	}

	// 创建 RocketMQ 生产者
	producer, err := rocketmq.NewTransactionProducer(
		NewTransactionListener(),
		producer.WithNsResolver(primitive.NewPassthroughResolver([]string{"127.0.0.1:9876"})),
		producer.WithRetry(1),
	)
	if err != nil {
		fmt.Printf("Failed to create producer: %v\n", err)

	}

	err = producer.Start()
	if err != nil {
		fmt.Printf("Failed to start producer: %v\n", err)

	}

	// 模拟订单支付信息
	orderID := "支付信息"
	msg := primitive.NewMessage("OrderPaidTopic", []byte(orderID))
	msg.WithDelayTimeLevel(5)

	// 发送半事务消息
	_, err = producer.SendMessageInTransaction(context.Background(), msg)
	if err != nil {
		fmt.Printf("Failed to send half message: %v\n", err)
	}

	// 关闭生产者
	producer.Shutdown()

	tx.Commit()
	return OrderInfoResponse, nil
}

func NewTransactionListener() *TransactionListener {
	return &TransactionListener{
		localTrans: new(sync.Map),
	}
}

// 事务监听器
type TransactionListener struct {
	localTrans       *sync.Map
	transactionIndex int32
}

func (tl *TransactionListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	// 在本地执行事务，可以在这里更新数据库等操作
	// 这里只是简单示例，直接返回事务提交状态
	return primitive.CommitMessageState
}

func (tl *TransactionListener) CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState {
	// 检查本地事务状态，这里可以查询数据库等操作
	// 假设订单支付成功返回事务提交状态，否则返回回滚状态
	return primitive.CommitMessageState
}

func (o OrderServer) OrderList(ctx context.Context, req *proto.OrderFilterRequest) (*proto.OrderListResponse, error) {
	//定义一个切片的变量 用来保存订单信息
	var orders []model.OrderInfo
	//定义返回结果集
	var rsp proto.OrderListResponse
	//订单总条数
	var total int64
	//通过gorm去查询用户的订单的总条数
	global.MysqlConf.DB.Model(&model.OrderInfo{User: req.UserId}).Count(&total)
	//global.DB.Table("orderinfo").Where(&model.OrderInfo{User: req.UserId}).Count(&total)
	//global.DB.Raw("select count(*) from orderinfo where user = ?", req.UserId).Scan(&total)
	rsp.Total = int32(total)

	//分页
	//select * from orderinfo where user=1 limit 1,10
	global.MysqlConf.DB.Scopes(Paginate(int(req.Pages), int(req.PagePerNums))).Where(&model.OrderInfo{User: req.UserId}).Find(&orders)

	//组装返回的订单数据
	for _, order := range orders {
		rsp.Data = append(rsp.Data, &proto.OrderInfoResponse{
			Id:      order.ID,
			UserId:  order.User,
			OrderSn: order.OrderSn,
			PayType: order.PayType,
			Status:  order.Status,
			Post:    order.Post,
			Total:   order.OrderMount,
			Address: order.Address,
			Name:    order.SignerName,
			Mobile:  order.SingerMobile,
			AddTime: order.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &rsp, nil
}

func (o OrderServer) OrderDetail(ctx context.Context, req *proto.OrderRequest) (*proto.OrderInfoDetailResponse, error) {
	var order model.OrderInfo
	var rsp proto.OrderInfoDetailResponse

	//这个订单的id是否是当前用户的订单， 如果在web层用户传递过来一个id的订单， web层应该先查询一下订单id是否是当前用户的
	//在个人中心可以这样做，但是如果是后台管理系统，web层如果是后台管理系统 那么只传递order的id，如果是电商系统还需要一个用户的id
	if result := global.MysqlConf.DB.Where(&model.OrderInfo{BaseModel: model.BaseModel{ID: req.Id}, User: req.UserId}).First(&order); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "订单不存在")
	}

	//如果订单查询出来，返回订单信息
	orderInfo := proto.OrderInfoResponse{}
	orderInfo.Id = order.ID
	orderInfo.UserId = order.User
	orderInfo.OrderSn = order.OrderSn
	orderInfo.PayType = order.PayType
	orderInfo.Status = order.Status
	orderInfo.Post = order.Post
	orderInfo.Total = order.OrderMount
	orderInfo.Address = order.Address
	orderInfo.Name = order.SignerName
	orderInfo.Mobile = order.SingerMobile

	rsp.OrderInfo = &orderInfo

	//如果一个订单是多个商品，我要知道是那几个商品，所以说定义一个切片，用来保存该订单下所有商品信息
	var orderGoods []model.OrderGoods
	if result := global.MysqlConf.DB.Where(&model.OrderGoods{Order: order.ID}).Find(&orderGoods); result.Error != nil {
		return nil, result.Error
	}

	for _, orderGood := range orderGoods {
		rsp.Goods = append(rsp.Goods, &proto.OrderItemResponse{
			GoodsId:    orderGood.Goods,
			GoodsName:  orderGood.GoodsName,
			GoodsPrice: orderGood.GoodsPrice,
			GoodsImage: orderGood.GoodsImage,
			Nums:       orderGood.Nums,
		})
	}

	return &rsp, nil
}

func (o OrderServer) OrderDetailBySn(ctx context.Context, req *proto.OrderDetailBySnRequest) (*proto.OrderInfoDetailResponse, error) {
	var order model.OrderInfo
	var rsp proto.OrderInfoDetailResponse

	//这个订单的id是否是当前用户的订单， 如果在web层用户传递过来一个id的订单， web层应该先查询一下订单id是否是当前用户的
	//在个人中心可以这样做，但是如果是后台管理系统，web层如果是后台管理系统 那么只传递order的id，如果是电商系统还需要一个用户的id
	if result := global.MysqlConf.DB.Where(&model.OrderInfo{OrderSn: req.OrderSn}).First(&order); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "订单不存在")
	}

	//如果订单查询出来，返回订单信息
	orderInfo := proto.OrderInfoResponse{}
	orderInfo.Id = order.ID
	orderInfo.UserId = order.User
	orderInfo.OrderSn = order.OrderSn
	orderInfo.PayType = order.PayType
	orderInfo.Status = order.Status
	orderInfo.Post = order.Post
	orderInfo.Total = order.OrderMount
	orderInfo.Address = order.Address
	orderInfo.Name = order.SignerName
	orderInfo.Mobile = order.SingerMobile

	rsp.OrderInfo = &orderInfo

	//如果一个订单是多个商品，我要知道是那几个商品，所以说定义一个切片，用来保存该订单下所有商品信息
	var orderGoods []model.OrderGoods
	if result := global.MysqlConf.DB.Where(&model.OrderGoods{Order: order.ID}).Find(&orderGoods); result.Error != nil {
		return nil, result.Error
	}

	for _, orderGood := range orderGoods {
		rsp.Goods = append(rsp.Goods, &proto.OrderItemResponse{
			GoodsId:    orderGood.Goods,
			GoodsName:  orderGood.GoodsName,
			GoodsPrice: orderGood.GoodsPrice,
			GoodsImage: orderGood.GoodsImage,
			Nums:       orderGood.Nums,
		})
	}

	return &rsp, nil
}

func (o OrderServer) UpdateOrderStatus(ctx context.Context, req *proto.OrderStatus) (*emptypb.Empty, error) {
	//先查询，再更新 实际上有两条sql执行， select 和 update语句
	var order = model.OrderInfo{}
	res := global.MysqlConf.DB.Model(&model.OrderInfo{}).Where("order_sn = ?", req.OrderSn).First(&order)
	if res.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "订单不存在")
	}
	order.Status = strconv.Itoa(int(req.Status))
	order.PayType = strconv.Itoa(int(req.PayType))
	order.TradeNo = req.TradeNo
	result := global.MysqlConf.DB.Save(&order)

	if result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "更新订单失败")
	}
	return &emptypb.Empty{}, nil
}

func GenerateOrderSn(userId int32) string {
	//订单号的生成规则
	/*
		年月日时分秒+用户id+2位随机数
	*/
	now := time.Now()
	rand.Seed(time.Now().UnixNano())
	orderSn := fmt.Sprintf("%d%d%d%d%d%d%d%d",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Nanosecond(),
		userId, rand.Intn(90)+10,
	)
	return orderSn
}
func Paginate(page, size int) func(db *gorm.DB) *gorm.DB {
	// 定义查询作用域
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(size).Offset((page - 1) * size)
	}
}

// 延迟队列
// 1.订单状态修改
// 2.库存归还
func OrderTimeOut(ctx context.Context, msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
	for i := range msgs {
		orderInfo := model.OrderInfo{}
		json.Unmarshal(msgs[i].Body, &orderInfo)
		tx := global.MysqlConf.DB.Begin()
		order := model.OrderInfo{}
		result := tx.Model(model.OrderInfo{}).Where(model.OrderInfo{OrderSn: orderInfo.OrderSn}).First(&order)
		if result.RowsAffected == 0 {
			return consumer.ConsumeRetryLater, nil
		}

		//判断是否支付成功
		if order.Status != "1" {
			order.Status = "4"
		}
		res := tx.Save(&order)
		if res.Error != nil {
			return consumer.ConsumeRetryLater, nil
		}

		msg := primitive.NewMessage("ReBackStock", msgs[i].Body)
		global.RocketMqProducer.SendSync(context.Background(), msg)
		commit := tx.Commit()
		if commit.Error == nil {
			return consumer.ConsumeSuccess, nil
		}
	}

	//在RocketMQ中，这几个常量代表了消费者处理消息时可能的不同结果：
	//ConsumeSuccess：表示消息成功被消费，消费者成功处理了该条消息。
	//ConsumeRetryLater：表示消费者暂时无法处理该消息，但希望稍后重新尝试消费。通常在遇到某些可恢复的错误时会选择这个选项，以便稍后再次尝试处理消息。
	//Commit：表示消息被成功处理，并且消费者确认了已经成功处理该消息，消息可以被标记为已消费。
	//Rollback：表示消费者无法处理该消息，需要将消息回滚到之前的状态，以便稍后再次尝试消费。通常在遇到无法恢复的错误或者需要重新处理的情况下选择这个选项。
	//SuspendCurrentQueueAMoment：表示暂时挂起当前队列一段时间，可能是因为消费者当前的处理能力不足以处理更多的消息，需要等待一段时间后再继续消费。
	return consumer.ConsumeSuccess, nil
}
