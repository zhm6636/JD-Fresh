package model

type EsGoods struct {
	ID         int32 `json:"id"`
	CategoryID int32 `json:"category_id"`
	BrandsID   int32 `json:"brands_id"`
	OnSale     int8  `json:"on_sale"`
	ShipFree   int8  `json:"ship_free"`
	IsNew      int8  `json:"is_new"`
	IsHot      int8  `json:"is_hot"`

	Name        string  `json:"name"`
	ClickNum    int32   `json:"click_num"`
	SoldNum     int32   `json:"sold_num"`
	FavNum      int32   `json:"fav_num"`
	MarketPrice float32 `json:"market_price"`
	GoodsBrief  string  `json:"goods_brief"`
	ShopPrice   float32 `json:"shop_price"`
}

// 接口多态
func (EsGoods) GetIndexName() string {
	return "goods"
}

func (EsGoods) GetMapping() string {
	goodsMapping := `
	{
		"mappings" : {
			"properties" : {
				"brands_id" : {
					"type" : "integer"
				},
				"category_id" : {
					"type" : "integer"
				},
				"click_num" : {
					"type" : "integer"
				},
				"fav_num" : {
					"type" : "integer"
				},
				"id" : {
					"type" : "integer"
				},
				"is_hot" : {
					"type" : "integer"
				},
				"is_new" : {
					"type" : "integer"
				},
				"market_price" : {
					"type" : "float"
				},
				"name" : {
					"type" : "text",
					"analyzer":"ik_max_word"
				},
				"goods_brief" : {
					"type" : "text",
					"analyzer":"ik_max_word"
				},
				"on_sale" : {
					"type" : "integer"
				},
				"ship_free" : {
					"type" : "integer"
				},
				"shop_price" : {
					"type" : "float"
				},
				"sold_num" : {
					"type" : "long"
				}
			}
		}
	}`
	return goodsMapping
}
