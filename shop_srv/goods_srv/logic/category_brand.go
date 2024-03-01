package logic

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"goods_srv/global"
	"goods_srv/model"
	"goods_srv/proto"
)

func (g GoodsServer) CategoryBrandList(ctx context.Context, req *proto.CategoryBrandFilterRequest) (*proto.CategoryBrandListResponse, error) {
	var categoryBrands []model.GoodsCategoryBrand
	categoryBrandListResponse := proto.CategoryBrandListResponse{}
	var total int64
	global.MysqlConf.DB.Model(&model.GoodsCategoryBrand{}).Count(&total)
	categoryBrandListResponse.Total = int32(total)
	global.MysqlConf.DB.Scopes(Paginate(int(req.Pages), int(req.PagePerNums))).Find(&categoryBrands)
	var categoryResponses []*proto.CategoryBrandResponse
	for _, categoryBrand := range categoryBrands {
		categoryResponses = append(categoryResponses, &proto.CategoryBrandResponse{
			Category: &proto.CategoryInfoResponse{
				Id:             categoryBrand.Category.ID,
				Name:           categoryBrand.Category.Name,
				Level:          categoryBrand.Category.Level,
				IsTab:          categoryBrand.Category.IsTab,
				ParentCategory: categoryBrand.Category.ParentCategoryID,
			},
			Brand: &proto.BrandInfoResponse{
				Id:   categoryBrand.Brands.ID,
				Name: categoryBrand.Brands.Name,
				Logo: categoryBrand.Brands.Logo,
			},
		})
	}
	categoryBrandListResponse.Data = categoryResponses
	return &categoryBrandListResponse, nil
}

func (g GoodsServer) GetCategoryBrandList(ctx context.Context, req *proto.CategoryInfoRequest) (*proto.BrandListResponse, error) {
	brandListResponse := proto.BrandListResponse{}
	var category model.Category
	if result := global.MysqlConf.DB.Find(&category, req.Id).First(&category); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌分类不存在")
	}
	var categoryBrands []model.GoodsCategoryBrand
	if result := global.MysqlConf.DB.Where(&model.GoodsCategoryBrand{CategoryID: category.ID}).Find(&categoryBrands); result.RowsAffected > 0 {
		brandListResponse.Total = int32(result.RowsAffected)
	}
	var brandInfoResponses []*proto.BrandInfoResponse
	for _, categoryBrand := range categoryBrands {
		brandInfoResponses = append(brandInfoResponses, &proto.BrandInfoResponse{
			Id:   int32(categoryBrand.Brands.ID),
			Name: categoryBrand.Brands.Name,
			Logo: categoryBrand.Brands.Logo,
		})
	}
	brandListResponse.Data = brandInfoResponses
	return &brandListResponse, nil
}

func (g GoodsServer) CreateCategoryBrand(ctx context.Context, req *proto.CategoryBrandRequest) (*proto.CategoryBrandResponse, error) {
	var category model.Category
	if result := global.MysqlConf.DB.First(&category, req.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	var brand model.Brands
	if result := global.MysqlConf.DB.First(&brand, req.BrandId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}
	categoryBrand := model.GoodsCategoryBrand{
		CategoryID: req.CategoryId,
		BrandsID:   req.BrandId,
	}
	global.MysqlConf.DB.Save(&categoryBrand)
	return &proto.CategoryBrandResponse{Id: int32(categoryBrand.ID)}, nil
}

func (g GoodsServer) DeleteCategoryBrand(ctx context.Context, req *proto.CategoryBrandRequest) (*proto.Empty, error) {
	if result := global.MysqlConf.DB.Delete(&model.GoodsCategoryBrand{}, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "品牌分类不存在")
	}
	return &proto.Empty{}, nil
}

func (g GoodsServer) UpdateCategoryBrand(ctx context.Context, req *proto.CategoryBrandRequest) (*proto.Empty, error) {
	var categoryBrand model.GoodsCategoryBrand
	if result := global.MysqlConf.DB.First(&categoryBrand, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌分类不存在")
	}
	var category model.Category
	if result := global.MysqlConf.DB.First(&category, req.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	var brand model.Brands
	if result := global.MysqlConf.DB.First(&brand, req.BrandId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}
	categoryBrand.CategoryID = req.CategoryId
	categoryBrand.BrandsID = req.BrandId
	global.MysqlConf.DB.Save(&categoryBrand)
	return &proto.Empty{}, nil
}
func Paginate(page, size int) func(db *gorm.DB) *gorm.DB {
	// 定义查询作用域
	return func(db *gorm.DB) *gorm.DB {
		return db.Limit(size).Offset((page - 1) * size)
	}
}
