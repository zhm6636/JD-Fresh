package logic

import (
	"context"
	"strconv"

	"user_srv/model"
	"user_srv/proto"
)

// GetUserList 获取用户列表
func (s *UserServer) GetUserList(c context.Context, in *proto.PageInfo) (*proto.UserListResponse, error) {

	list, err := model.GetUserList(int(in.Pn), int(in.PSize))
	if err != nil {
		return nil, err
	}
	var userListResponse []*proto.UserInfoResponse
	for _, user := range list {
		userListResponse = append(userListResponse, newUserInfoResponse(user))
	}
	return &proto.UserListResponse{
		Total: int32(len(userListResponse)),
		Data:  userListResponse,
	}, nil
}

// newUserInfoResponse 构建响应
func newUserInfoResponse(data *model.User) *proto.UserInfoResponse {
	return &proto.UserInfoResponse{
		Id:       int32(data.ID),
		PassWord: data.Password,
		NickName: data.Nickname,
		Mobile:   data.Mobile,
		BirthDay: uint64(data.Birthday.Unix()),
		Gender:   strconv.Itoa(data.Gander),
		Role:     int32(data.Role),
	}
}
