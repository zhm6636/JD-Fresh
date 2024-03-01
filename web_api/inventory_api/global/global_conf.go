package global

import "gorm.io/gorm"

type MysqlConfig struct {
	DB       *gorm.DB
	Addr     string
	Port     int
	User     string
	Password string
	Dsn      string
	DataBase string
}
