package api

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
)

// configJsonBody json request body.
type configJsonBody struct {
	Id            string
	CaptchaType   string
	VerifyValue   string
	DriverAudio   *base64Captcha.DriverAudio
	DriverString  *base64Captcha.DriverString
	DriverChinese *base64Captcha.DriverChinese
	DriverMath    *base64Captcha.DriverMath
	DriverDigit   *base64Captcha.DriverDigit
}

var store = base64Captcha.DefaultMemStore

// base64Captcha create return id, b64s, err
func GetCaptcha() (string, string, string, error) {

	// https://captcha.mojotv.cn/ 调试配置
	var param configJsonBody = configJsonBody{
		Id:          "",
		CaptchaType: "string",
		VerifyValue: "",
		DriverAudio: &base64Captcha.DriverAudio{},
		DriverString: &base64Captcha.DriverString{
			Length:          4,
			Height:          60,
			Width:           240,
			ShowLineOptions: 2,
			NoiseCount:      0,
			Source:          "12iD3XhCQfYF5sf6FaMrzrGFxzrKJ4u85L",
		},
		DriverChinese: &base64Captcha.DriverChinese{},
		DriverMath:    &base64Captcha.DriverMath{},
		DriverDigit:   &base64Captcha.DriverDigit{},
	}
	var driver base64Captcha.Driver

	//create base64 encoding captcha
	switch param.CaptchaType {
	case "audio":
		driver = param.DriverAudio
	case "string":
		driver = param.DriverString.ConvertFonts()
	case "math":
		driver = param.DriverMath.ConvertFonts()
	case "chinese":
		driver = param.DriverChinese.ConvertFonts()
	default:
		driver = param.DriverDigit
	}
	c := base64Captcha.NewCaptcha(driver, store)
	return c.Generate()
	// id, b64s, err := c.Generate()

	// body := map[string]interface{}{"code": 1, "data": b64s, "captchaId": id, "msg": "success"}
	// if err != nil {
	// 	body = map[string]interface{}{"code": 0, "msg": err.Error()}
	// }
	// var _ = body

	// // log.Println(body)
	// log.Println(1)
	// log.Println(id)

	// log.Printf("store =%+v\n", store)
}

// base64Captcha verify
func VerifyCaptcha(id, VerifyValue string) bool {
	return store.Verify(id, VerifyValue, true)
}

func TestMyCaptcha() {
	id, b64s, _, err := GetCaptcha()
	if err != nil {
		return
	}
	var _ = b64s
	log.Println("id =", id)
	log.Println("VerifyValue =", store.Get(id, true))
	result := VerifyCaptcha(id, store.Get(id, true))
	log.Println("result =", result)
}

// CreateCaptcha 生成图形验证码
func CreateCaptcha(c *gin.Context) {
	id, b64s, _, err := GetCaptcha()
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
		"data":    b64s,
		"id":      id,
	})

}
