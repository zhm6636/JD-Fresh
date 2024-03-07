package model

import (
	"database/sql/driver"
	"encoding/json"
)

// 单个商品的库存详情结构体
type GoodsDetail struct {
	Goods int32
	Num   int32
}
type GoodsDetailList []GoodsDetail

func (g GoodsDetailList) Value() (driver.Value, error) {
	return json.Marshal(g)
}

// 实现 sql.Scanner 接口，Scan 将 value 扫描至 Jsonb
func (g *GoodsDetailList) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &g)
}

// 定义库存表
type Inventory struct {
	BaseModel
	Goods   int32 `gorm:"type:int;index"` //商品id
	Stocks  int32 `gorm:"type:int"`       //库存数量
	Version int32 `gorm:"type:int"`       //分布式锁的乐观锁
}

// 库存扣减历史表
type StockSellDetail struct {
	OrderSn string          `gorm:"type:varchar(200);index:idx_order_sn,unique;"`
	Status  int32           `gorm:"type:varchar(200)"` //1 表示已扣减 2. 表示已归还
	Detail  GoodsDetailList `gorm:"type:varchar(200)"`
}
