package logic

import (
	"context"
	"strconv"

	"user_srv/model"
	"user_srv/proto"
)

// GetUserById 根据获取用户信息
func (s *UserServer) GetUserById(c context.Context, in *proto.IdRequest) (*proto.UserInfoResponse, error) {
	id, err := model.GetUserById(int(in.Id))
	if err != nil {
		return nil, err
	}
	return &proto.UserInfoResponse{
		Id:       int32(id.ID),
		PassWord: id.Password,
		NickName: id.Nickname,
		Mobile:   id.Mobile,
		BirthDay: uint64(id.Birthday.Unix()),
		Gender:   strconv.Itoa(id.Gander),
		Role:     int32(id.Role),
	}, nil
}
