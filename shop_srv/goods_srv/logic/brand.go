package logic

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"goods_srv/global"
	"goods_srv/model"
	"goods_srv/proto"
)

func (g GoodsServer) BrandList(ctx context.Context, req *proto.BrandFilterRequest) (*proto.BrandListResponse, error) {
	brandListResponse := proto.BrandListResponse{}
	var brands []model.Brands
	result := global.MysqlConf.DB.Scopes(Paginate(int(req.Pages), int(req.PagePerNums))).Find(&brands)
	if result.Error != nil {
		return nil, result.Error
	}
	var total int64
	global.MysqlConf.DB.Model(&model.Brands{}).Count(&total)
	brandListResponse.Total = int32(total)
	var brandResponses []*proto.BrandInfoResponse
	for _, brand := range brands {
		brandResponses = append(brandResponses, &proto.BrandInfoResponse{
			Id:   brand.ID,
			Name: brand.Name,
			Logo: brand.Logo,
		})
	}
	brandListResponse.Data = brandResponses
	return &brandListResponse, nil
}

func (g GoodsServer) CreateBrand(ctx context.Context, req *proto.BrandRequest) (*proto.BrandInfoResponse, error) {
	if result := global.MysqlConf.DB.First(&model.Brands{}); result.RowsAffected == 1 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌已存在")
	}
	brand := &model.Brands{
		Name: req.Name,
		Logo: req.Logo,
	}
	global.MysqlConf.DB.Save(brand)
	return &proto.BrandInfoResponse{Id: brand.ID}, nil
}

func (g GoodsServer) DeleteBrand(ctx context.Context, req *proto.BrandRequest) (*proto.Empty, error) {
	if result := global.MysqlConf.DB.Delete(&model.Brands{}, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "品牌不存在")
	}
	return &proto.Empty{}, nil
}

func (g GoodsServer) UpdateBrand(ctx context.Context, req *proto.BrandRequest) (*proto.Empty, error) {
	brands := model.Brands{}
	if result := global.MysqlConf.DB.First(&brands); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}
	if req.Name != "" {
		brands.Name = req.Name
	}
	if req.Logo != "" {
		brands.Logo = req.Logo
	}
	global.MysqlConf.DB.Save(&brands)
	return &proto.Empty{}, nil
}
