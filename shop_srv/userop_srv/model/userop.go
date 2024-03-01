package model

const (
	LEAVING_MESSAGES = iota + 1
	COMPLAINT
	INQUIRY
	POST_SALE
	WANT_TO_BUY
)

type LeavingMessages struct {
	BaseModel
	//用户id,留言人信息
	User int32 `gorm:"type:int;index"`
	//留言类型
	MessageType int32 `gorm:"type:int comment '留言类型: 1(留言),2(投诉),3(询问),4(售后),5(求购)'"`
	//主题
	Subject string `gorm:"type:varchar(100)"`
	//留言内容
	Message string
	//图片地址
	File string `gorm:"type:varchar(200)"`
}

func (LeavingMessages) TableName() string {
	return "leavingmessages"
}

type Address struct {
	BaseModel
	//用户id
	User int32 `gorm:"type:int;index"`
	//省
	Province string `gorm:"type:varchar(10)"`
	//市
	City string `gorm:"type:varchar(10)"`
	//区/县
	District string `gorm:"type:varchar(20)"`
	//详细地址
	Address string `gorm:"type:varchar(100)"`
	//收件人名字
	SignerName string `gorm:"type:varchar(20)"`
	//收件人的手机号
	SignerMobile string `gorm:"type:varchar(11)"`
}

type UserFav struct {
	BaseModel
	//用户id
	User int32 `gorm:"type:int;index:idx_user_goods,unique"`
	//商品id
	Goods int32 `gorm:"type:int;index:idx_user_goods,unique"`
}

func (UserFav) TableName() string {
	return "userfav"
}
