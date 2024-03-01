package verify

// PassWordLoginForm 密码登录表单
type PassWordLoginForm struct {
	Mobile    string `form:"mobile" json:"mobile" binding:"required,mobile"` //手机号码格式有规范可寻， 自定义validator
	PassWord  string `form:"password" json:"password" binding:"required,min=3,max=20"`
	Captcha   string `form:"captcha" json:"captcha" binding:"required,min=4,max=5"`
	CaptchaId string `form:"captchaId" json:"captchaId" binding:"required"`
}
