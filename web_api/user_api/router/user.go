package router

import (
	"user_api/api"
	"user_api/middlewares"

	"github.com/gin-gonic/gin"
)

func InitUserRouter(Router *gin.RouterGroup) {
	UserRouter := Router.Group("user")
	{
		UserRouter.GET("", api.GetUserList)
		UserRouter.POST("/login", api.MobileLogin)
		UserRouter.POST("register", api.Register)

		//UserRouter.GET("detail", middlewares.JWTAuth(), api.GetUserDetail)
		UserRouter.PATCH("update", middlewares.JWTAuth(), api.Update)
	}
	//服务注册和发现
}
