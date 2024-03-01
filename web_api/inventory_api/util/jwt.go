package util

import (
	"strconv"

	"github.com/golang-jwt/jwt/v5"

	"inventory_api/global"
)

// CreateToken 创建token
func CreateToken(userid uint) (string, error) {
	key := global.JWT
	claims := jwt.MapClaims{
		"userid": userid,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(key))
}

// AuthToken 验证token 并验证userid是否正确
func AuthToken(token string, userid string) error {
	userId, err := strconv.Atoi(userid)
	if err != nil {
		return err
	}
	key := global.JWT
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})
	if err != nil {
		return err
	}
	claims := t.Claims.(jwt.MapClaims)
	if uint(claims["userid"].(float64)) != uint(userId) {
		return err
	}
	return nil
}
