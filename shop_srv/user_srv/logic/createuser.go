package logic

import (
	"context"
	"strconv"

	"user_srv/model"
	"user_srv/proto"
)

// CreateUser 创建用户
func (s *UserServer) CreateUser(c context.Context, in *proto.CreateUserInfo) (*proto.UserInfoResponse, error) {

	//if !util2.VerifyMobile(in.Mobile) {
	//	return nil, errors.New("手机号格式错误")
	//}
	//
	//if !util2.VerifyPassword(in.PassWord) {
	//	return nil, errors.New("密码格式错误")
	//}

	user, err := model.CreateUser(&model.User{
		Mobile:   in.Mobile,
		Password: model.MakePassword(in.PassWord),
		Nickname: in.NickName,
		Birthday: in.Birthday.AsTime(),
	})
	if err != nil {
		return nil, err
	}
	return &proto.UserInfoResponse{
		Id:       int32(user.ID),
		PassWord: user.Password,
		NickName: user.Nickname,
		Mobile:   user.Mobile,
		BirthDay: uint64(user.Birthday.Unix()),
		Gender:   strconv.Itoa(user.Gander),
		Role:     int32(user.Role),
	}, nil
}
