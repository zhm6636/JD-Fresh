package logic

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"goods_srv/global"
	"goods_srv/model"
	"goods_srv/proto"
)

func (g GoodsServer) BannerList(ctx context.Context, empty *proto.Empty) (*proto.BannerListResponse, error) {
	bannerListResponse := proto.BannerListResponse{}
	var banners []model.Banner
	result := global.MysqlConf.DB.Find(&banners)
	bannerListResponse.Total = int32(result.RowsAffected)
	var bannerReponses []*proto.BannerResponse
	for _, banner := range banners {
		bannerReponses = append(bannerReponses, &proto.BannerResponse{
			Id:    banner.ID,
			Image: banner.Image,
			Index: banner.Index,
			Url:   banner.Url,
		})
	}
	bannerListResponse.Data = bannerReponses
	return &bannerListResponse, nil
}

func (g GoodsServer) CreateBanner(ctx context.Context, req *proto.BannerRequest) (*proto.BannerResponse, error) {
	banner := model.Banner{}
	banner.Image = req.Image
	banner.Index = req.Index
	banner.Url = req.Url
	global.MysqlConf.DB.Save(&banner)
	return &proto.BannerResponse{Id: banner.ID}, nil
}

func (g GoodsServer) DeleteBanner(ctx context.Context, req *proto.BannerRequest) (*proto.Empty, error) {
	if result := global.MysqlConf.DB.Delete(&model.Banner{}, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "轮播图不存在")
	}
	return &proto.Empty{}, nil
}

func (g GoodsServer) UpdateBanner(ctx context.Context, req *proto.BannerRequest) (*proto.Empty, error) {
	var banner model.Banner
	if result := global.MysqlConf.DB.First(&banner, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "轮播图不存在")
	}
	if req.Url != "" {
		banner.Url = req.Url
	}
	if req.Image != "" {
		banner.Image = req.Image
	}
	if req.Index != 0 {
		banner.Index = req.Index
	}
	global.MysqlConf.DB.Save(&banner)
	return &proto.Empty{}, nil
}
