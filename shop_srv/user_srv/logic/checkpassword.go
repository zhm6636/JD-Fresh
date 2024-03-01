package logic

import (
	"context"

	"user_srv/model"
	"user_srv/proto"
)

// CheckPassWord 检查密码
func (s *UserServer) CheckPassWord(c context.Context, in *proto.PasswordCheckInfo) (*proto.CheckResponse, error) {
	//password := model.MakePassword(in.Password)

	//if !util.VerifyPassword(in.Password) {
	//	return nil, errors.New("密码格式错误")
	//}

	verifyPassword := model.VerifyPassword(in.Password, in.EncryptedPassword)
	return &proto.CheckResponse{
		Success: verifyPassword,
	}, nil
}
