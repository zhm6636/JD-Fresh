package logic

import (
	"context"

	"userop_srv/global"
	"userop_srv/model"
	"userop_srv/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetAddressList 获取地址列表
func (*UserOpServer) GetAddressList(ctx context.Context, req *proto.AddressRequest) (*proto.AddressListResponse, error) {
	var addresses []model.Address
	var rsp proto.AddressListResponse
	var addressResponse []*proto.AddressResponse

	if result := global.MysqlConf.DB.Where(&model.Address{User: req.UserId}).Find(&addresses); result.RowsAffected != 0 {
		rsp.Total = int32(result.RowsAffected)
	}

	for _, address := range addresses {
		addressResponse = append(addressResponse, &proto.AddressResponse{
			Id:           address.ID,
			UserId:       address.User,
			Province:     address.Province,
			City:         address.City,
			District:     address.District,
			Address:      address.Address,
			SignerName:   address.SignerName,
			SignerMobile: address.SignerMobile,
		})
	}
	rsp.Data = addressResponse

	return &rsp, nil
}

// CreateAddress 创建地址
func (*UserOpServer) CreateAddress(ctx context.Context, req *proto.AddressRequest) (*proto.AddressResponse, error) {
	var address model.Address

	address.User = req.UserId
	address.Province = req.Province
	address.City = req.City
	address.District = req.District
	address.Address = req.Address
	address.SignerName = req.SignerName
	address.SignerMobile = req.SignerMobile

	global.MysqlConf.DB.Save(&address)

	return &proto.AddressResponse{Id: address.ID}, nil
}

// DeleteAddress 删除地址
func (*UserOpServer) DeleteAddress(ctx context.Context, req *proto.AddressRequest) (*emptypb.Empty, error) {
	if result := global.MysqlConf.DB.Where("id=? and user=?", req.Id, req.UserId).Delete(&model.Address{}); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "收货地址不存在")
	}
	return &emptypb.Empty{}, nil
}

// UpdateAddress 修改地址
func (*UserOpServer) UpdateAddress(ctx context.Context, req *proto.AddressRequest) (*emptypb.Empty, error) {
	var address model.Address

	if result := global.MysqlConf.DB.Where("id=? and user=?", req.Id, req.UserId).First(&address); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "收获地址不存在")
	}

	if address.Province != "" {
		address.Province = req.Province
	}

	if address.City != "" {
		address.City = req.City
	}

	if address.District != "" {
		address.District = req.District
	}

	if address.Address != "" {
		address.Address = req.Address
	}

	if address.SignerName != "" {
		address.SignerName = req.SignerName
	}

	if address.SignerMobile != "" {
		address.SignerMobile = req.SignerMobile
	}

	global.MysqlConf.DB.Save(&address)

	return &emptypb.Empty{}, nil
}
