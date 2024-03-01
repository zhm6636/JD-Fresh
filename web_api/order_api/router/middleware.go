package router

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"order_api/util"
)

// AuthToken token验证
func AuthToken(c *gin.Context) {
	zap.S().Infof("auth token")
	token := c.GetHeader("token")
	userid := c.GetHeader("userid")
	err := util.AuthToken(token, userid)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    500,
			"message": "token验证失败",
		})
		c.Abort()
	}

	c.Next()
}

// AuthTokenUse token验证
func AuthTokenUse(Router ...*gin.RouterGroup) {
	for k, v := range Router {
		v.Use(AuthToken)
		zap.S().Debugf("auth token use router %v", k)
	}
}
