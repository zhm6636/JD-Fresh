package model

import (
	"log"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"inventory_srv/global"
)

var db *gorm.DB

// 初始化数据库 迁移
func init() {
	db = global.MysqlConf.DB
	// 自动迁移 (这是GORM自动创建表的一种方式--译者注)
	//db.NamingStrategy = schema.NamingStrategy{
	//	//TablePrefix:   "t_", // 表前缀
	//	SingularTable: true, // 表名单数
	//	//NoLowerCase:   true, //跳过蛇形命名
	//}

	err := db.AutoMigrate(&Inventory{}, StockSellDetail{})
	if err != nil {
		zap.S().Panic(err)
	}

	if err != nil {
		log.Panicln(err)
	}
}
