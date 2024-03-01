package model

import (
	"log"

	"gorm.io/gorm"

	"user_srv/global"
)

var db *gorm.DB

// 初始化数据库 迁移
func init() {
	db = global.MysqlConf.DB
	err := db.AutoMigrate(&User{})
	if err != nil {
		log.Panicln(err)
	}
}
