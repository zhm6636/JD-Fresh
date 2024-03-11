package logic

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
	"github.com/opentracing/opentracing-go"
	"go.uber.org/zap"
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "Order_Srv_Create")
	defer span.Finish()
	/*
	   新建订单
	       1. 从购物车中获取到选中的商品
	       2. 商品的价格自己查询 - 访问商品服务 (跨微服务)
	       3. 库存的扣减 - 访问库存服务 (跨微服务)
	       4. 订单的基本信息表 - 订单的商品信息表
	       5. 从购物车中删除已购买的记录
	*/
	//半消息事务的监听着
	orderListener := TransactionListener{Ctx: ctx}

	p, err := rocketmq.NewTransactionProducer(
		&orderListener,
		producer.WithGroupName(global.Nacos["rocketmq"].(map[string]interface{})["rebacktopic"].(string)),
		producer.WithNameServer([]string{fmt.Sprintf("%s:%d", global.Nacos["rocketmq"].(map[string]interface{})["host"].(string), global.Nacos["rocketmq"].(map[string]interface{})["port"].(int))}),
	)
	defer p.Shutdown()
	if err != nil {
		zap.S().Errorf("生成producer失败: %s", err.Error())
		return nil, err
	}

	if err = p.Start(); err != nil {
		zap.S().Errorf("启动producer失败: %s", err.Error())
		return nil, err
	}

	order := model.OrderInfo{
		//订单编号一定在创建订单生成，确保归还库存的一致性
		OrderSn:      GenerateOrderSn(req.UserId),
		Address:      req.Address,
		SignerName:   req.Name,
		SingerMobile: req.Mobile,
		Post:         req.Post,
		User:         req.UserId,
		Status:       "0",
	}
	//应该在消息中具体指明一个订单的具体的商品的扣减情况
	jsonString, _ := json.Marshal(order)

	//半事务消息把消息放到的是归还库存的队列
	_, err = p.SendMessageInTransaction(context.Background(),
		primitive.NewMessage(fmt.Sprintf("%s", global.Nacos["rocketmq"].(map[string]interface{})["rebacktopic"].(string)), jsonString))
	if err != nil {
		fmt.Printf("发送失败: %s\n", err)
		return nil, status.Error(codes.Internal, "发送消息失败")
	}

	if orderListener.Code != codes.OK {
		return nil, status.Error(orderListener.Code, orderListener.Detail)
	}

	return &proto.OrderInfoResponse{Id: orderListener.ID, OrderSn: order.OrderSn, Total: orderListener.OrderAmount}, nil
}

func NewTransactionListener() *TransactionListener {
	return &TransactionListener{}
}

// 事务监听器
type TransactionListener struct {
	Code        codes.Code
	Detail      string
	ID          int32
	OrderAmount float32
	Ctx         context.Context
}

// ExecuteLocalTransaction 这里操作本地事务，做下订单，订单商品和删除购物车
func (o *TransactionListener) ExecuteLocalTransaction(msg *primitive.Message) primitive.LocalTransactionState {
	span, ctx := opentracing.StartSpanFromContext(o.Ctx, "Order_Srv_ExecuteLocalTransaction")
	defer span.Finish()
	var orderInfo model.OrderInfo
	//字符串转结构体
	_ = json.Unmarshal(msg.Body, &orderInfo)
	var goodsIds []int32
	var shopCarts []model.ShoppingCart
	goodsNumsMap := make(map[int32]int32)

	ShoppingCart := opentracing.StartSpan("Goods_Srv_ShoppingCart", opentracing.ChildOf(span.Context()))
	if result := global.MysqlConf.DB.Where(&model.ShoppingCart{User: orderInfo.User, Checked: true}).Find(&shopCarts); result.RowsAffected == 0 {
		o.Code = codes.InvalidArgument
		o.Detail = "没有选中结算的商品"
		return primitive.RollbackMessageState
	}
	ShoppingCart.Finish()

	for _, shopCart := range shopCarts {
		goodsIds = append(goodsIds, shopCart.Goods)
		goodsNumsMap[shopCart.Goods] = shopCart.Nums

	}

	//跨服务调用商品微服务
	BatchGetGoods := opentracing.StartSpan("Goods_Srv_BatchGetGoods", opentracing.ChildOf(span.Context()))
	res, err := global.GoodsClient.BatchGetGoods(ctx, &goods.BatchGoodsIdInfo{Id: goodsIds})
	if err != nil {
		o.Code = codes.Internal
		o.Detail = "批量查询商品信息失败"
		return primitive.RollbackMessageState
	}
	BatchGetGoods.Finish()

	var orderAmount float32
	var orderGoods []*model.OrderGoods
	var goodsInvInfo []*inventory.GoodsInvInfo
	for _, good := range res.Data {
		orderAmount += good.ShopPrice * float32(goodsNumsMap[good.Id])
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

	//跨服务调用库存微服务进行库存扣减
	/*
		1. 调用库存服务的trysell
		2. 调用仓库服务的trysell
		3. 调用积分服务的tryAdd
		任何一个服务出现了异常，那么你得调用对应的所有的微服务的cancel接口
		如果所有的微服务都正常，那么你得调用所有的微服务的confirm
	*/

	Sell := opentracing.StartSpan("Inventory_Srv_TrySell", opentracing.ChildOf(span.Context()))
	if _, err = global.InventoryClient.Sell(ctx, &inventory.SellInfo{OrderSn: orderInfo.OrderSn, GoodsInfo: goodsInvInfo}); err != nil {
		//如果是因为网络问题， 这种如何避免误判， 大家自己改写一下sell的返回逻辑
		o.Code = codes.ResourceExhausted
		o.Detail = "扣减库存失败"
		return primitive.CommitMessageState
	}
	Sell.Finish()

	//生成订单表
	//20210308xxxx
	CreateorderInfo := opentracing.StartSpan("Goods_Srv_CreateOrderInfo", opentracing.ChildOf(span.Context()))
	tx := global.MysqlConf.DB.Begin()
	orderInfo.OrderMount = orderAmount
	if result := tx.Save(&orderInfo); result.RowsAffected == 0 {
		tx.Rollback()
		o.Code = codes.Internal
		o.Detail = "创建订单失败"
		return primitive.CommitMessageState
	}
	CreateorderInfo.Finish()

	o.OrderAmount = orderAmount
	o.ID = orderInfo.ID
	for _, orderGood := range orderGoods {
		orderGood.Order = orderInfo.ID
	}

	//批量插入orderGoods

	CreateInBatches := opentracing.StartSpan("Goods_Srv_CreateInBatches", opentracing.ChildOf(span.Context()))
	if result := tx.CreateInBatches(orderGoods, 100); result.RowsAffected == 0 {
		tx.Rollback()
		o.Code = codes.Internal
		o.Detail = "批量插入订单商品失败"
		return primitive.CommitMessageState
	}
	CreateInBatches.Finish()

	DeleteShoppingCart := opentracing.StartSpan("Goods_Srv_DeleteShoppingCart", opentracing.ChildOf(span.Context()))
	if result := tx.Where(&model.ShoppingCart{User: orderInfo.User, Checked: true}).Delete(&model.ShoppingCart{}); result.RowsAffected == 0 {
		tx.Rollback()
		o.Code = codes.Internal
		o.Detail = "删除购物车记录失败"
		return primitive.CommitMessageState
	}
	DeleteShoppingCart.Finish()

	//发送延时消息
	p, err := rocketmq.NewProducer(producer.WithGroupName(global.Nacos["rocketmq"].(map[string]interface{})["timeoutgroup"].(string)), producer.WithNameServer([]string{fmt.Sprintf("%s:%d", global.Nacos["rocketmq"].(map[string]interface{})["host"].(string), global.Nacos["rocketmq"].(map[string]interface{})["port"].(int))}))
	if err != nil {
		panic("生成producer失败:" + err.Error())
	}
	defer p.Shutdown()

	//不要在一个进程中使用多个producer， 但是不要随便调用shutdown因为会影响其他的producer
	if err = p.Start(); err != nil {
		panic("启动producer失败")
	}

	msg = primitive.NewMessage(global.Nacos["rocketmq"].(map[string]interface{})["timeouttopic"].(string), msg.Body)
	msg.WithDelayTimeLevel(4)
	_, err = p.SendSync(context.Background(), msg)
	if err != nil {
		zap.S().Errorf("发送延时消息失败: %v\n", err)
		tx.Rollback()
		o.Code = codes.Internal
		o.Detail = "发送延时消息失败"
		return primitive.CommitMessageState
	}
	zap.S().Infof("发送消息成功")

	//提交事务
	tx.Commit()
	o.Code = codes.OK
	return primitive.RollbackMessageState
}

func (o *TransactionListener) CheckLocalTransaction(msg *primitive.MessageExt) primitive.LocalTransactionState {
	var orderInfo model.OrderInfo
	_ = json.Unmarshal(msg.Body, &orderInfo)

	//怎么检查之前的逻辑是否完成
	if result := global.MysqlConf.DB.Where(model.OrderInfo{OrderSn: orderInfo.OrderSn}).First(&orderInfo); result.RowsAffected == 0 {
		return primitive.CommitMessageState //你并不能说明这里就是库存已经扣减了
	}

	return primitive.RollbackMessageState
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
	span, ctx := opentracing.StartSpanFromContext(ctx, "Order_Srv_OrderTimeOut")
	defer span.Finish()
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

		msg := primitive.NewMessage(global.Nacos["rocketmq"].(map[string]interface{})["rebacktopic"].(string), msgs[i].Body)
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

//docker run -d --name consul --network=kong-net --ip=192.168.0.3 -v D:/consul.d:/consul/config.d -p 8500:8500 -p 8600:8600 -p 8600:8600/udp -p 8300:8300 -p 8301:8301 -p 8301:8301/udp -p 8302:8302 -p 8302:8302/udp consul agent -server -bootstrap -ui -config-dir=/consul/config.d -client=0.0.0.0
//docker run -d --name kong-database --network=kong-net --ip=192.168.0.4 -p 5432:5432 -e "POSTGRES_USER=kong" -e "POSTGRES_DB=kong" -e "POSTGRES_PASSWORD=kong" postgres:9.6
//docker run --rm --network=kong-net -e "KONG_DATABASE=postgres" -e "KONG_PG_HOST=kong-database" -e "KONG_PG_USER=kong" -e "KONG_PG_PASSWORD=kong" -e "KONG_CASSANDRA_CONTACT_POINTS=kong-database" kong:3.6.1 kong migrations bootstrap
//docker run -d --name kong --network=kong-net --ip=192.168.0.5 -e "KONG_DATABASE=postgres" -e "KONG_PG_HOST=192.168.0.4" -e "KONG_PG_USER=kong" -e "KONG_PG_PASSWORD=kong" -e "KONG_PROXY_ACCESS_LOG=/dev/stdout" -e "KONG_ADMIN_ACCESS_LOG=/dev/stdout" -e "KONG_PROXY_ERROR_LOG=/dev/stderr" -e "KONG_ADMIN_ERROR_LOG=/dev/stderr" -e "KONG_ADMIN_LISTEN=0.0.0.0:8001, 0.0.0.0:8444 ssl" -e "KONG_PROXY_LISTEN=0.0.0.0:8000, 0.0.0.0:9080 http2, 0.0.0.0:9081 http2 ssl" -e "KONG_DNS_RESOLVER=192.168.0.3:8600" -e "KONG_DNS_ORDER=SRV,LAST,A,CNAME" -p 8000:8000 -p 9080:9080 -p 8443:8443 -p 8001:8001 -p 127.0.0.1:8444:8444 kong:3.6.1
//docker run -d --name konga -p 1337:1337 --network kong-net pantsel/konga
//8001: 管理端api http访问端口
//8444: 管理端api ssl访问端口
//8000：http访问端口(http)
//8443：http访问端口(ssl)
//9080: grpc端口(http2)
//docker run -d --name jaeger -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 -p 6831:6831/udp -p 6832:6832/udp -p 5778:5778 -p 16686:16686 -p 4317:4317 -p 4318:4318 -p 14250:14250 -p 14268:14268 -p 14269:14269 -p 9411:9411 -e SPAN_STORAGE_TYPE=elasticsearch -e ES_ARCHIVE_SERVER_URLS="http://42.192.108.133:9200" -e ES_SERVER_URLS="http://42.192.108.133:9200" -e ES_USERNAME=elastic -e ES_PASSWORD=Zhm5833366..  -e ES_NODES_WAN_ONLY=true -v D:/certs:/usr/local/openjdk-8/lib/security/cacerts jaegertracing/all-in-one:1.55
//"D:/ca.crt"
//docker run -d --name jaeger -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 -p 6831:6831/udp -p 6832:6832/udp -p 5778:5778 -p 16686:16686 -p 4317:4317 -p 4318:4318 -p 14250:14250 -p 14268:14268 -p 14269:14269 -p 9411:9411 jaegertracing/all-in-one:1.55
