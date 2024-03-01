package logic

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"userop_srv/global"
	"userop_srv/model"
	"userop_srv/proto"
)

// GetFavList 获取收藏列表
func (*UserOpServer) GetFavList(ctx context.Context, req *proto.UserFavRequest) (*proto.UserFavListResponse, error) {
	var rsp proto.UserFavListResponse
	var userFavs []model.UserFav
	var userFavList []*proto.UserFavResponse
	//查询用户的收藏记录
	//查询某件商品被哪些用户收藏了
	result := global.MysqlConf.DB.Where(&model.UserFav{User: req.UserId, Goods: req.GoodsId}).Find(&userFavs)
	rsp.Total = int32(result.RowsAffected)

	for _, userFav := range userFavs {
		userFavList = append(userFavList, &proto.UserFavResponse{
			UserId:  userFav.User,
			GoodsId: userFav.Goods,
		})
	}

	rsp.Data = userFavList

	return &rsp, nil
}

// AddUserFav 收藏
func (*UserOpServer) AddUserFav(ctx context.Context, req *proto.UserFavRequest) (*emptypb.Empty, error) {
	var userFav model.UserFav

	userFav.User = req.UserId
	userFav.Goods = req.GoodsId

	global.MysqlConf.DB.Save(&userFav)

	return &emptypb.Empty{}, nil
}

// DeleteUserFav 删除收藏
func (*UserOpServer) DeleteUserFav(ctx context.Context, req *proto.UserFavRequest) (*emptypb.Empty, error) {
	if result := global.MysqlConf.DB.Unscoped().Where("goods=? and user=?", req.GoodsId, req.UserId).Delete(&model.UserFav{}); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "收藏记录不存在")
	}
	return &emptypb.Empty{}, nil
}

// GetUserFavDetail 获取收藏详情
func (*UserOpServer) GetUserFavDetail(ctx context.Context, req *proto.UserFavRequest) (*emptypb.Empty, error) {
	var userfav model.UserFav
	if result := global.MysqlConf.DB.Where("goods=? and user=?", req.GoodsId, req.UserId).Find(&userfav); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "收藏记录不存在")
	}
	return &emptypb.Empty{}, nil
}
