package logic

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"goods_srv/global"
	"goods_srv/model"
	"goods_srv/proto"
)

func (g GoodsServer) GetAllCategoryList(ctx context.Context, empty *proto.Empty) (*proto.CategoryListResponse, error) {
	var categorys []model.Category
	global.MysqlConf.DB.Where(&model.Category{Level: 1}).Preload("SubCategory.SubCategory").Find(&categorys)
	b, _ := json.Marshal(&categorys)
	return &proto.CategoryListResponse{JsonData: string(b)}, nil
}

func (g GoodsServer) CreateCategory(ctx context.Context, req *proto.CategoryInfoRequest) (*proto.CategoryInfoResponse, error) {
	category := model.Category{}
	category.Name = req.Name
	category.Level = req.Level
	if req.Level != 1 {
		category.ParentCategoryID = req.ParentCategory
	}
	category.IsTab = req.IsTab
	res := global.MysqlConf.DB.Save(&category)
	if res.Error != nil {
		zap.S().Errorf("插入数据失败:%s", res.Error.Error())
	}
	return &proto.CategoryInfoResponse{Id: int32(category.ID)}, nil
}

func (g GoodsServer) DeleteCategory(ctx context.Context, req *proto.DeleteCategoryRequest) (*proto.Empty, error) {
	if result := global.MysqlConf.DB.Delete(&model.Category{}, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "商品分类不存在")
	}
	return &proto.Empty{}, nil
}

func (g GoodsServer) UpdateCategory(ctx context.Context, req *proto.CategoryInfoRequest) (*proto.Empty, error) {
	var category model.Category
	if result := global.MysqlConf.DB.First(&category, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "商品分类不存在")
	}
	if req.Name != "" {
		category.Name = req.Name
	}
	if req.ParentCategory != 0 {
		category.ParentCategoryID = req.ParentCategory
	}
	if req.Level != 0 {
		category.Level = req.Level
	}
	if req.IsTab {
		category.IsTab = req.IsTab
	}
	global.MysqlConf.DB.Save(&category)
	return &proto.Empty{}, nil
}
