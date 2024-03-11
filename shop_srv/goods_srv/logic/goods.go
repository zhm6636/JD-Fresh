package logic

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/olivere/elastic/v7"
	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"goods_srv/global"
	"goods_srv/model"
	"goods_srv/proto"
)

func (g GoodsServer) GoodsList(ctx context.Context, req *proto.GoodsFilterRequest) (*proto.GoodsListResponse, error) {
	span, ctx := opentracing.StartSpanFromContext(ctx, "Goods_Srv_List")
	defer span.Finish()
	//关键词搜索、查询新品、查询热门商品、通过价格区间筛选， 通过商品分类筛选
	goodsListResponse := &proto.GoodsListResponse{}

	//match bool 复合查询
	q := elastic.NewBoolQuery()

	//搜索的关键词
	if req.KeyWords != "" {
		//必须满足的条件 商品名字和商品介绍都要包含关键字
		q = q.Must(elastic.NewMultiMatchQuery(req.KeyWords, "name", "goods_brief"))
	}

	//是否是热门
	if req.IsHot == 1 {
		//在es中链式操作 拼接是否是热门
		q = q.Filter(elastic.NewTermQuery("is_hot", req.IsHot))
	}

	//是否是新品
	if req.IsNew == 1 {
		q = q.Filter(elastic.NewTermQuery("is_new", req.IsNew))
	}

	//商品最低价格
	if req.PriceMin > 0 {
		//Gte >=
		q = q.Filter(elastic.NewRangeQuery("shop_price").Gte(req.PriceMin))
	}

	//商品的最大价格
	if req.PriceMax > 0 {
		//Lte <=
		q = q.Filter(elastic.NewRangeQuery("shop_price").Lte(req.PriceMax))
	}

	//拼接商品品牌条件
	if req.Brand > 0 {
		q = q.Filter(elastic.NewTermQuery("brands_id", req.Brand))
	}

	//通过category去查询商品
	var subQuery string
	categoryIds := make([]interface{}, 0)

	//拼接分类的条件,假如1级分类要查包含二级分类的商品数据
	if req.TopCategory > 0 {
		var category model.Category
		if result := global.MysqlConf.DB.First(&category, req.TopCategory); result.RowsAffected == 0 {
			return nil, status.Errorf(codes.NotFound, "商品分类不存在")
		}

		if category.Level == 1 {
			//一级分类所有的分类id
			subQuery = fmt.Sprintf("select id from category where parent_category_id in (select id from category WHERE parent_category_id=%d)", req.TopCategory)
		} else if category.Level == 2 {
			subQuery = fmt.Sprintf("select id from category WHERE parent_category_id=%d", req.TopCategory)
		} else if category.Level == 3 {
			subQuery = fmt.Sprintf("select id from category WHERE id=%d", req.TopCategory)
		}

		type Result struct {
			ID int32
		}
		var results []Result
		global.MysqlConf.DB.Model(model.Category{}).Raw(subQuery).Scan(&results)
		for _, re := range results {
			categoryIds = append(categoryIds, re.ID)
		}

		//生成terms查询
		q = q.Filter(elastic.NewTermsQuery("category_id", categoryIds...))
	}
	//return nil, status.Errorf(codes.NotFound, "商品分类不存在")
	//当前页
	if req.Pages == 0 {
		req.Pages = 1
	}

	//每页显示条数
	switch {
	case req.PagePerNums > 100:
		req.PagePerNums = 100
	case req.PagePerNums <= 0:
		req.PagePerNums = 10
	}

	//es查询
	//Index 索引 类似mysql数据库
	//偏移量计算 （当前页-1 ）* 每页显示条数
	offset := (int(req.Pages) - 1) * int(req.PagePerNums)
	result, err := global.EsClient.Search().Index(model.EsGoods{}.GetIndexName()).Query(q).From(offset).Size(int(req.PagePerNums)).Do(context.Background())
	if err != nil {
		return nil, err
	}

	if len(result.Hits.Hits) == 0 {
		return nil, status.Errorf(codes.NotFound, "没有商品了")
	}

	//只要es中的商品id
	goodsIds := make([]int32, 0)
	//总条是对的
	goodsListResponse.Total = int32(result.Hits.TotalHits.Value)
	for _, value := range result.Hits.Hits {
		goods := model.EsGoods{}
		_ = json.Unmarshal(value.Source, &goods)
		goodsIds = append(goodsIds, goods.ID)
	}

	//回表查询
	//查询id在某个数组中的值
	var goods []model.Goods
	//这个相当于 select * from goods  where id in ()
	//Preload("Category") select * from category
	//Preload("Brands") select * from brand
	re := global.MysqlConf.DB.Model(model.Goods{}).Preload("Category").Preload("Brands").Find(&goods, goodsIds)
	if re.Error != nil {
		return nil, re.Error
	}

	for _, good := range goods {
		goodsInfoResponse := ModelToResponse(&good)
		goodsListResponse.Data = append(goodsListResponse.Data, goodsInfoResponse)
	}

	return goodsListResponse, nil

}

func (g GoodsServer) BatchGetGoods(ctx context.Context, req *proto.BatchGoodsIdInfo) (*proto.GoodsListResponse, error) {
	goodsListResponse := &proto.GoodsListResponse{}
	var goods []model.Goods
	//调用where并不会真正执行sql 只是用来生成sql的 当调用find， first才会去执行sql，
	result := global.MysqlConf.DB.Where(req.Id).Find(&goods)
	for _, good := range goods {
		goodsInfoResponse := ModelToResponse(&good)
		goodsListResponse.Data = append(goodsListResponse.Data, goodsInfoResponse)
	}
	goodsListResponse.Total = int32(result.RowsAffected)
	return goodsListResponse, nil
}

func (g GoodsServer) CreateGoods(ctx context.Context, req *proto.CreateGoodsInfo) (*proto.GoodsInfoResponse, error) {
	var category model.Category
	if result := global.MysqlConf.DB.First(&category, req.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	var brand model.Brands
	if result := global.MysqlConf.DB.First(&brand, req.BrandId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}
	//这里没有看到图片文件是如何上传， 在微服务中 普通的文件上传已经不再使用
	goods := model.Goods{
		Brands:          brand,
		BrandsID:        brand.ID,
		Category:        category,
		CategoryID:      category.ID,
		Name:            req.Name,
		GoodsSn:         req.GoodsSn,
		MarketPrice:     req.MarketPrice,
		ShopPrice:       req.ShopPrice,
		GoodsBrief:      req.GoodsBrief,
		ShipFree:        int8(req.ShipFree),
		Images:          req.Images,
		DescImages:      req.DescImages,
		GoodsFrontImage: req.GoodsFrontImage,
		IsNew:           int8(req.IsNew),
		IsHot:           int8(req.IsHot),
		OnSale:          int8(req.OnSale),
	}
	global.MysqlConf.DB.Save(&goods)
	return &proto.GoodsInfoResponse{
		Id: goods.ID,
	}, nil
}

func (g GoodsServer) DeleteGoods(ctx context.Context, req *proto.DeleteGoodsInfo) (*proto.Empty, error) {
	if result := global.MysqlConf.DB.Delete(&model.Goods{}, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "商品不存在")
	}
	return &proto.Empty{}, nil
}

func (g GoodsServer) UpdateGoods(ctx context.Context, req *proto.CreateGoodsInfo) (*proto.Empty, error) {
	var goods model.Goods
	if result := global.MysqlConf.DB.First(&goods, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品不存在")
	}
	var category model.Category
	if result := global.MysqlConf.DB.First(&category, req.CategoryId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "商品分类不存在")
	}
	var brand model.Brands
	if result := global.MysqlConf.DB.First(&brand, req.BrandId); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "品牌不存在")
	}
	goods.Brands = brand
	goods.Category = category
	goods.Name = req.Name
	goods.GoodsSn = req.GoodsSn
	goods.MarketPrice = req.MarketPrice
	goods.ShopPrice = req.ShopPrice
	goods.GoodsBrief = req.GoodsBrief
	goods.ShipFree = int8(req.ShipFree)
	goods.Images = req.Images
	goods.DescImages = req.DescImages
	goods.GoodsFrontImage = req.GoodsFrontImage
	goods.IsNew = int8(req.IsNew)
	goods.IsHot = int8(req.IsHot)
	goods.OnSale = int8(req.OnSale)
	global.MysqlConf.DB.Save(&goods)
	return &proto.Empty{}, nil
}

func (g GoodsServer) GetGoodsDetail(ctx context.Context, req *proto.GoodInfoRequest) (*proto.GoodsInfoResponse, error) {
	var goods model.Goods
	if result := global.MysqlConf.DB.First(&goods, req.Id); result.RowsAffected == 0 {
		return nil, status.Errorf(codes.NotFound, "商品不存在")
	}
	goodsInfoResponse := ModelToResponse(&goods)
	return goodsInfoResponse, nil
}

func ModelToResponse(goods *model.Goods) *proto.GoodsInfoResponse {
	return &proto.GoodsInfoResponse{
		Id:              goods.ID,
		CategoryId:      goods.CategoryID,
		Name:            goods.Name,
		GoodsSn:         goods.GoodsSn,
		ClickNum:        goods.ClickNum,
		SoldNum:         goods.SoldNum,
		FavNum:          goods.FavNum,
		MarketPrice:     goods.MarketPrice,
		ShopPrice:       goods.ShopPrice,
		GoodsBrief:      goods.GoodsBrief,
		ShipFree:        int32(goods.ShipFree),
		GoodsFrontImage: goods.GoodsFrontImage,
		IsNew:           int32(goods.IsNew),
		IsHot:           int32(goods.IsHot),
		OnSale:          int32(goods.OnSale),
		DescImages:      goods.DescImages,
		Images:          goods.Images,
		Category: &proto.CategoryBriefInfoResponse{
			Id:   goods.Category.ID,
			Name: goods.Category.Name,
		},
		Brand: &proto.BrandInfoResponse{
			Id:   goods.Brands.ID,
			Name: goods.Brands.Name,
			Logo: goods.Brands.Logo,
		},
	}
}
