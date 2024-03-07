package router

import (
	"github.com/gin-gonic/gin"

	"order_api/api/order"
	"order_api/api/pay"
	"order_api/middlewares"
)

func InitOrderRouter(Router *gin.RouterGroup) {
	OrderRouter := Router.Group("orders").Use(middlewares.JWTAuth())
	{
		OrderRouter.GET("", order.List)       // 订单列表
		OrderRouter.POST("", order.New)       // 新建订单
		OrderRouter.GET("/:id", order.Detail) // 订单详情

	}
	PayRouter := Router.Group("/pay")
	{
		PayRouter.POST("", middlewares.JWTAuth(), pay.AliPay) // 支付连接
		PayRouter.POST("/alipay/notify", pay.NotifyUrl)
		PayRouter.GET("/alipay/return", pay.ReturnUrl)
	}
}
