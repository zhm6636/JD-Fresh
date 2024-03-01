package model

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"

	"github.com/anaskhan96/go-password-encoder"
	"gorm.io/gorm"
)

// BaseModel 模型基类
type BaseModel struct {
	ID        uint           `gorm:"primarykey"`
	CreatedAt time.Time      `gorm:"column:add_time;autoCreateTime"`
	UpdatedAt time.Time      `gorm:"column:update_time;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
	IsDeleted bool           `gorm:"column:is_deleted"`
}

// User 用户模型
type User struct {
	BaseModel
	Mobile   string    `gorm:"type:varchar(15);not null;unique;index:idx_mobile;comment:手机号" json:"mobile"`
	Password string    `gorm:"type:varchar(200);not null;comment:密码" json:"password"`
	Nickname string    `gorm:"type:varchar(20);not null;comment:昵称" json:"nickname"`
	Avatar   string    `gorm:"type:varchar(200);not null;comment:头像" json:"avatar"`
	Birthday time.Time `gorm:"type:datetime;not null;comment:生日" json:"birthday"`
	Gander   int       `gorm:"type:tinyint;not null;default:1;comment:性别(1:男;2:女)" json:"gander"`
	Role     int       `gorm:"type:tinyint;not null;default:1;comment:角色(1:普通用户;2:管理员)" json:"role"`
}

// GetTableName 获取表名
func GetUserList(page, size int) ([]*User, error) {
	var users []*User
	err := db.Where("is_deleted = ?", false).Offset((page - 1) * size).Limit(size).Find(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}

// GetUserById 根据id获取用户信息
func GetUserById(id int) (*User, error) {
	var user User
	err := db.Where("is_deleted = ?", false).First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByMobile 根据手机号获取用户信息
func GetUserByMobile(mobile string) (*User, error) {
	var user User
	err := db.Where("is_deleted = ?", false).First(&user, "mobile = ?", mobile).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// CreateUser 创建用户
func CreateUser(u *User) (*User, error) {
	err := db.Create(u).Error
	return u, err
}

// UpdateUser 更新用户
func UpdateUser(u *User) error {
	return db.Save(u).Error
}

// 加密密码的方法
func MakePassword(plainPwd string) string {
	var options = &password.Options{10, 10000, 50, md5.New}
	salt, encodedPwd := password.Encode(plainPwd, options)
	//我们在数据库存储的密码是什么样的呢？
	newPassword := fmt.Sprintf("$pbkdf2-sha512$%s$%s", salt, encodedPwd) // pbkdf2-sha512 $符号拼接
	return newPassword
}

// 验证密码的方法
func VerifyPassword(plainPwd, encodedPwd string) bool {
	options := &password.Options{10, 10000, 50, md5.New}
	split := strings.Split(encodedPwd, "$")
	verify := password.Verify(plainPwd, split[2], split[3], options)
	return verify
}
