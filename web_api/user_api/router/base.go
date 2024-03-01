package router

import (
	"github.com/gin-gonic/gin"

	"user_api/api"
)

func InitBaseRouter(Router *gin.RouterGroup) {
	BaseRouter := Router.Group("base")
	{
		BaseRouter.GET("captcha", api.CreateCaptcha)
		//BaseRouter.POST("send_sms", api.SendSms)
	}

}
