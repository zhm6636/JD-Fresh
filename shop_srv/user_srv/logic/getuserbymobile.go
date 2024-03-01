package logic

import (
	"context"
	"strconv"

	"user_srv/model"
	"user_srv/proto"
)

// GetUserByMobile 根据手机号获取用户信息
func (s *UserServer) GetUserByMobile(c context.Context, in *proto.MobileRequest) (*proto.UserInfoResponse, error) {
	//if !util2.VerifyMobile(in.Mobile) {
	//	return nil, errors.New("手机号格式错误")
	//}

	mobile, err := model.GetUserByMobile(in.Mobile)
	if err != nil {
		return nil, err
	}
	return &proto.UserInfoResponse{
		Id:       int32(mobile.ID),
		PassWord: mobile.Password,
		NickName: mobile.Nickname,
		Mobile:   mobile.Mobile,
		BirthDay: uint64(mobile.Birthday.Unix()),
		Gender:   strconv.Itoa(mobile.Gander),
		Role:     int32(mobile.Role),
	}, nil
}
