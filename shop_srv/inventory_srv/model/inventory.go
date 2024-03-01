package model

// 定义库存表
type Inventory struct {
	BaseModel
	Goods   int32 `gorm:"type:int;index"` //商品id
	Stocks  int32 `gorm:"type:int"`       //库存数量
	Version int32 `gorm:"type:int"`       //分布式锁的乐观锁
}
