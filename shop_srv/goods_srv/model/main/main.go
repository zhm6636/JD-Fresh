package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/olivere/elastic/v7"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"goods_srv/model"
)

func main() {
	var err error
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,   // Slow SQL threshold
			LogLevel:                  logger.Silent, // Log level
			IgnoreRecordNotFoundError: true,          // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      true,          // Don't include params in the SQL log
			Colorful:                  true,          // Disable color
		},
	)
	db, err := gorm.Open(mysql.Open("root:Zhm5833366..@tcp(42.192.108.133:3306)/test?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Panicln(err)
	}
	db.AutoMigrate(&model.Category{}, &model.Brands{}, &model.Banner{}, &model.GoodsCategoryBrand{}, &model.Goods{})
	if err != nil {
		log.Panicln(err)
	}

	certFile := "./conf/elasticsearch.crt"
	key := "./conf/elasticsearch.key"

	// Load certificate
	cert, err := tls.LoadX509KeyPair(certFile, key)
	if err != nil {
		log.Panic(err)
	}

	// Create a custom TLS configuration
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, // Disable server certificate verification
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			// Implement custom verification logic here
			return nil
		},
	}
	addr := "42.192.108.133"
	port := 9200
	url := fmt.Sprintf("https://%s:%d", addr, port)
	EsClient, err := elastic.NewClient(elastic.SetURL(url), elastic.SetSniff(false), elastic.SetHealthcheck(true), elastic.SetBasicAuth("elastic", "Zhm5833366.."), elastic.SetHttpClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}))
	if err != nil {
		panic(err)
	}
	//_, err = EsClient.CreateIndex("goods").BodyString(model.EsGoods{}.GetMapping()).Do(context.Background())
	//if err != nil {
	//	panic(err)
	//}

	var goods []model.Goods
	res := db.Model(&model.Goods{}).Find(&goods)
	if res.Error != nil {
		log.Panicln(res.Error)
	}
	fmt.Println("mysql goods:", goods)

	for _, v := range goods {
		esgoods := model.EsGoods{
			ID:          v.ID,
			CategoryID:  v.CategoryID,
			BrandsID:    v.BrandsID,
			OnSale:      v.OnSale,
			ShipFree:    v.ShipFree,
			IsNew:       v.IsNew,
			IsHot:       v.IsHot,
			Name:        v.Name,
			ClickNum:    v.ClickNum,
			SoldNum:     v.SoldNum,
			FavNum:      v.FavNum,
			MarketPrice: v.MarketPrice,
			GoodsBrief:  v.GoodsBrief,
			ShopPrice:   v.ShopPrice,
		}
		fmt.Println("开始插入es第", esgoods.ID, "条数据")
		do, err := EsClient.Index().Index("goods").Id(strconv.Itoa(int(esgoods.ID))).BodyJson(esgoods).Do(context.Background())
		if err != nil {
			panic(err)
		}
		fmt.Println(do)
	}
}
