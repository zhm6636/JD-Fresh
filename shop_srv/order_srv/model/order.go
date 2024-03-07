package model

import "time"

// 购物车表
type ShoppingCart struct {
	BaseModel
	User    int32 `gorm:"type:int;index"`
	Goods   int32 `gorm:"type:int;index"`
	Nums    int32 `gorm:"type:int"`
	Checked bool
}

func (ShoppingCart) TableName() string {
	return "shoppingcart"
}

// 订单表
type OrderInfo struct {
	BaseModel
	User    int32  `gorm:"type:int;index"`
	OrderSn string `gorm:"type:varchar(30);index"`
	PayType string `gorm:"type:varchar(20) comment 'alipay(支付宝)， wechat(微信)'"`
	//status大家可以考虑使用iota来做
	Status       string `gorm:"type:varchar(20)  comment 'PAYING(待支付), TRADE_SUCCESS(成功)， TRADE_SUCCESS(超时关闭), WAIT_BUYER_PAY(交易创建), TRADE_FINISHED(交易结束)'"`
	TradeNo      string `gorm:"type:varchar(100) comment '交易号'"`
	OrderMount   float32
	PayTime      *time.Time
	Address      string `gorm:"type:varchar(100)"`
	SignerName   string `gorm:"type:varchar(20)"`
	SingerMobile string `gorm:"type:varchar(11)"`
	Post         string `gorm:"type:varchar(20)"`
}

func (OrderInfo) TableName() string {
	return "orderinfo"
}

// 订单商品表
type OrderGoods struct {
	BaseModel
	Order      int32  `gorm:"type:int;index"`
	Goods      int32  `gorm:"type:int;index"`
	GoodsName  string `gorm:"type:varchar(100);index"`
	GoodsImage string `gorm:"type:varchar(200)"`
	GoodsPrice float32
	Nums       int32 `gorm:"type:int"`
}

func (OrderGoods) TableName() string {
	return "ordergoods"
}
