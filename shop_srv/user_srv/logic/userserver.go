package logic

import "user_srv/proto"

// UserServer 用户服务
type UserServer struct {
	proto.UnimplementedUserServer
}
