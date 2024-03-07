package pay

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/smartwalle/alipay/v3"
	"go.uber.org/zap"

	"order_api/api"
	"order_api/global"
	"order_api/proto"
)

var (
	ALIPAY = 1
	WEIPAY = 2
)

// 生成支付宝方式
func AliPay(ctx *gin.Context) {
	//需不需要订单信息，订单详情
	id := ctx.PostForm("id")
	userId, _ := ctx.Get("userId")
	i, err := strconv.Atoi(id)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"msg": "url格式出错",
		})
		return
	}
	rsp, err := global.OrderClient.OrderDetail(context.Background(), &proto.OrderRequest{
		Id:     int32(i),
		UserId: int32(userId.(uint)),
	})
	if err != nil {
		zap.S().Errorw("获取订单详情失败")
		api.HandleGrpcErrorToHttp(err, ctx)
		return
	}

	//生成支付地址
	//生成支付宝的支付url
	client, err := alipay.New(global.Nacos["alipay"].(map[string]interface{})["AppID"].(string), global.Nacos["alipay"].(map[string]interface{})["PrivateKey"].(string), false)

	if err != nil {
		zap.S().Errorw("实例化支付宝失败")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"msg": err.Error(),
		})
		return
	}
	err = client.LoadAliPayPublicKey(global.Nacos["alipay"].(map[string]interface{})["AliPublicKey"].(string))
	if err != nil {
		zap.S().Errorw("加载支付宝的公钥失败")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"msg": err.Error(),
		})
		return
	}

	var p = alipay.TradePagePay{}
	p.NotifyURL = global.Nacos["alipay"].(map[string]interface{})["NotifyURL"].(string)
	p.ReturnURL = global.Nacos["alipay"].(map[string]interface{})["ReturnURL"].(string)
	p.Subject = "电商订单-" + rsp.OrderInfo.OrderSn
	p.OutTradeNo = rsp.OrderInfo.OrderSn
	p.TotalAmount = strconv.FormatFloat(float64(rsp.OrderInfo.Total), 'f', 2, 64)
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	url, err := client.TradePagePay(p)
	if err != nil {
		zap.S().Errorw("生成支付url失败")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"msg": err.Error(),
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"alipay_url": url.String(),
	})
}

// 同步回调
// 同步回调只需要做订单状态展示
// 查询
func ReturnUrl(ctx *gin.Context) {
	//接收所有的get请求
	orderSn := ctx.Query("out_trade_no")
	//根据订单号查询订单详情
	res, err := global.OrderClient.OrderDetailBySn(context.Background(), &proto.OrderDetailBySnRequest{
		OrderSn: orderSn,
	})
	if err != nil {
		zap.S().Errorw("订单状态查询失败")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"msg": err.Error(),
		})
		return
	}

	//然后返回
	ctx.JSON(http.StatusOK, gin.H{
		"code": 0,
		"msg":  "ok",
		"data": res,
	})
}

// 异步回调
// 异步回调是支付宝服务器请求你的异步回调地址，1.必须是post请求 2.必须是线上可以访问的地址 3.异步回调方法打印不了数据，是需要做日志记录
// 修改订单状态
func NotifyUrl(ctx *gin.Context) {
	//把你的所有rpc和api都部署上linux
	//把异步回调地址更改
	//实例化支付宝的客户端
	client, err := alipay.New(global.Nacos["alipay"].(map[string]interface{})["AppID"].(string), global.Nacos["alipay"].(map[string]interface{})["PrivateKey"].(string), false)
	if err != nil {
		zap.S().Errorw("实例化支付宝失败")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"msg": err.Error(),
		})
		return
	}
	//验证是否是支付宝请求，安全校验
	err = client.LoadAliPayPublicKey(global.Nacos["alipay"].(map[string]interface{})["AliPublicKey"].(string))
	if err != nil {
		zap.S().Errorw("加载支付宝的公钥失败")
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"msg": err.Error(),
		})
		return
	}

	//获取支付宝请求携带的参数
	params, err := client.GetTradeNotification(ctx.Request)
	zap.S().Info(params)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{})
		return
	}

	status := 0
	//根据订单号修改订单状态
	if string(params.TradeStatus) == "TRADE_SUCCESS" {
		status = 1
	} else if string(params.TradeStatus) == "TRADE_CLOSED" {
		status = 2
	} else if string(params.TradeStatus) == "TRADE_FINISHED" {
		status = 3
	} else {
		status = 4
	}

	//修改支付方式和修改流水号
	_, err = global.OrderClient.UpdateOrderStatus(context.Background(), &proto.OrderStatus{
		OrderSn: params.OutTradeNo,
		Status:  int32(status),
		PayType: int32(ALIPAY),
		TradeNo: params.TradeNo,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{})
		return
	}

	//如果订单状态更新成功，直接返回一个success字符串
	ctx.String(http.StatusOK, "success")
}
