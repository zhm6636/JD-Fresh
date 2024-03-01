package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goods_api/global"
	"goods_api/middlewares"
)

// InitRouter 初始化路由
func init() {
	r := gin.Default()
	// 注册健康检查路由
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	//配置跨域
	r.Use(middlewares.Cors())
	//添加链路追踪
	ApiGroup := r.Group("/g/v1")
	InitGoodsRouter(ApiGroup)
	InitCategoryRouter(ApiGroup)
	InitBannerRouter(ApiGroup)
	InitBrandRouter(ApiGroup)

	global.Router = r
}

// CORSMiddleware 设置跨域请求头
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		// 处理OPTIONS请求，不进入下一个中间件
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		// 继续处理请求
		c.Next()
	}
}
