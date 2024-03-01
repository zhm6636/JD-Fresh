package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"user_api/global"
	"user_api/middlewares"
	"user_api/proto"
	"user_api/verify"
)

// GetUserList 获取用户列表
func GetUserList(c *gin.Context) {
	page := c.Query("page")
	size := c.Query("size")
	p, _ := strconv.Atoi(page)
	l, _ := strconv.Atoi(size)
	//调用服务端的方法 获取用户列表
	list, err := global.UserClient.GetUserList(context.Background(), &proto.PageInfo{
		Pn:    uint32(p),
		PSize: uint32(l),
	})
	if err != nil {
		c.JSON(200, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"code":    200,
		"message": "success",
		"data":    list,
	})
}

// MobileLogin 手机号登录
func MobileLogin(c *gin.Context) {

	//表单验证
	passwordLoginForm := verify.PassWordLoginForm{}

	//gin去验证这个值是否正确
	err := c.ShouldBind(&passwordLoginForm)

	//去实现自定义验证器错误提示信息
	if err != nil {
		//处理err错误信息
		HandleValidatorError(c, err)
		return
	}

	mobile := passwordLoginForm.Mobile
	password := passwordLoginForm.PassWord
	captcha := passwordLoginForm.Captcha
	captchaId := passwordLoginForm.CaptchaId

	zap.S().Infof(captcha, captchaId)
	//if !VerifyCaptcha(captchaId, captcha) {
	//	c.JSON(200, gin.H{
	//		"code":    500,
	//		"message": "验证码错误",
	//	})
	//	return
	//}

	//if !util2.VerifyMobile(mobile) {
	//	c.JSON(200, gin.H{
	//		"code":    500,
	//		"message": "手机号格式错误",
	//	})
	//	return
	//}
	//
	//if !util2.VerifyPassword(password) {
	//	c.JSON(200, gin.H{
	//		"code":    500,
	//		"message": "密码格式错误",
	//	})
	//	return
	//}

	//调用服务端的方法 跟据手机号获取用户信息
	rsp, err := global.UserClient.GetUserByMobile(context.Background(), &proto.MobileRequest{
		Mobile: mobile,
	})
	if err != nil {
		c.JSON(200, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	//调用服务端的方法 校验密码
	word, err := global.UserClient.CheckPassWord(context.Background(), &proto.PasswordCheckInfo{
		Password:          password,
		EncryptedPassword: rsp.PassWord,
	})
	if err != nil {
		c.JSON(200, gin.H{
			"code":    500,
			"message": err.Error(),
		})
	}

	if word.Success == false {
		c.JSON(200, gin.H{
			"code":    500,
			"message": "密码错误",
		})
		return
	}

	//生成token
	j := middlewares.NewJWT()
	claims := middlewares.CustomClaims{
		ID:          uint(rsp.Id),
		NickName:    rsp.NickName,
		AuthorityId: uint(rsp.Role),
		StandardClaims: jwt.StandardClaims{
			NotBefore: time.Now().Unix(),               //签名的生效时间
			ExpiresAt: time.Now().Unix() + 60*60*24*30, //30天过期
			Issuer:    "shop",
		},
	}
	token, err := j.CreateToken(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "生成token失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         rsp.Id,
		"nick_name":  rsp.NickName,
		"token":      token,
		"expired_at": (time.Now().Unix() + 60*60*24*30) * 1000,
	})

}

// MobileRegister 手机号注册
func Register(c *gin.Context) {
	data, _ := c.GetRawData()
	var body map[string]string
	_ = json.Unmarshal(data, &body)

	//获取json中的key，注意使用["key"]获取
	mobile := body["mobile"]
	password := body["password"]
	nickname := body["nickname"]
	birthday := body["birthday"]

	//解析时间
	parse, err := time.Parse("2006-01-02", birthday)
	if err != nil {
		return
	}

	//调用服务端的方法 跟据手机号获取用户信息
	_, err = global.UserClient.GetUserByMobile(context.Background(), &proto.MobileRequest{
		Mobile: mobile,
	})
	//if err != nil {
	//	c.JSON(200, gin.H{
	//		"code":    500,
	//		"message": "用户已存在",
	//	})
	//	return
	//}

	//调用服务端的方法 创建用户
	user, err := global.UserClient.CreateUser(context.Background(), &proto.CreateUserInfo{
		Mobile:   mobile,
		PassWord: password,
		NickName: nickname,
		Birthday: timestamppb.New(parse),
	})
	if err != nil {
		return
	}

	//生成token
	j := middlewares.NewJWT()
	claims := middlewares.CustomClaims{
		ID:          uint(user.Id),
		NickName:    user.NickName,
		AuthorityId: uint(user.Role),
		StandardClaims: jwt.StandardClaims{
			NotBefore: time.Now().Unix(),               //签名的生效时间
			ExpiresAt: time.Now().Unix() + 60*60*24*30, //30天过期
			Issuer:    "shop",
		},
	}
	token, err := j.CreateToken(claims)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "生成token失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         user.Id,
		"nick_name":  user.NickName,
		"token":      token,
		"expired_at": (time.Now().Unix() + 60*60*24*30) * 1000,
	})
}

func Update(c *gin.Context) {
	captchaId := c.PostForm("captchaId")
	nickName := c.PostForm("nickName")
	gender := c.PostForm("gender")
	birthDay := c.PostForm("birthDay")
	parse, err := time.Parse("2006-01-02 15:04:05", birthDay)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    500,
			"message": "日期格式错误",
		})
		return
	}

	userId, err := strconv.Atoi(captchaId)
	if err != nil {
		c.JSON(200, gin.H{
			"code":    500,
			"message": "id格式错误",
		})
		return
	}

	//调用服务端的方法 更新用户信息
	_, err = global.UserClient.UpdateUser(context.Background(), &proto.UpdateUserInfo{
		Id:       int32(userId),
		NickName: nickName,
		Gender:   gender,
		BirthDay: uint64(parse.Unix()),
	})

	if err != nil {
		c.JSON(200, gin.H{
			"code":    500,
			"message": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"code":    200,
		"message": "success",
	})

}

// HandleValidatorError 处理验证器错误
func HandleValidatorError(c *gin.Context, err error) {
	errs, ok := err.(validator.ValidationErrors)
	if !ok {

		c.JSON(http.StatusOK, gin.H{
			"msg": err.Error(),
		})
		return
	}
	zap.S().Debugf("%v", removeTopStruct(errs.Translate(global.Trans)))
	c.JSON(http.StatusBadRequest, gin.H{
		//把错误自定义翻译器
		"error": removeTopStruct(errs.Translate(global.Trans)),
	})
	return
}

// removeTopStruct 去除掉错误提示中的结构体标识
func removeTopStruct(fileds map[string]string) map[string]string {
	rsp := map[string]string{}
	for field, err := range fileds {
		rsp[field[strings.Index(field, ".")+1:]] = err
	}
	return rsp
}
