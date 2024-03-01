package main

import (
	"log"
	"os"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"goods_srv/model"
)

func main() {
	var err error
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,          // Don't include params in the SQL log
			Colorful:                  true,          // Disable color
		},
	)
	db, err := gorm.Open(mysql.Open("root:root@tcp(127.0.0.1:3306)/go_web?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Panicln(err)
	}
	db.AutoMigrate(&model.Category{}, &model.Brands{}, &model.Banner{}, &model.GoodsCategoryBrand{}, &model.Goods{})
	if err != nil {
		log.Panicln(err)
	}
}
