package logic

import (
	"context"
	"strconv"
	"time"

	"user_srv/model"
	"user_srv/proto"
)

// UpdateUser 更新用户信息
func (s *UserServer) UpdateUser(c context.Context, in *proto.UpdateUserInfo) (*proto.Empty, error) {
	id, err := model.GetUserById(int(in.Id))
	if err != nil {
		return nil, err
	}
	atoi, _ := strconv.Atoi(in.Gender)
	timeObj := time.Unix(int64(in.BirthDay), 0)
	id.Nickname = in.NickName
	id.Gander = atoi
	id.Birthday = timeObj

	err = model.UpdateUser(id)
	if err != nil {
		return nil, err
	}
	return &proto.Empty{}, nil

}
