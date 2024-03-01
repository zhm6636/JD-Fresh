package router

import (
	"order_api/api/shop_cart"
	"order_api/middlewares"

	"github.com/gin-gonic/gin"
)

func InitShopCartRouter(Router *gin.RouterGroup) {
	GoodsRouter := Router.Group("shopcarts").Use(middlewares.JWTAuth())
	{
		GoodsRouter.GET("", shop_cart.List)          //购物车列表
		GoodsRouter.DELETE("/:id", shop_cart.Delete) //删除条目
		GoodsRouter.POST("", shop_cart.New)          //添加商品到购物车
		//PATCH 局部修改信息 PUT	全量修改商品信息
		GoodsRouter.PATCH("/:id", shop_cart.Update) //修改条目
	}
}
