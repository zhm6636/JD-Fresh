package main

import (
	"context"
	"fmt"
	"log"
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

	//certFile := "./conf/elasticsearch.crt"
	//key := "./conf/elasticsearch.key"

	// Load certificate
	//cert, err := tls.LoadX509KeyPair(certFile, key)
	//if err != nil {
	//	log.Panic(err)
	//}

	// Create a custom TLS configuration
	//tlsConfig := &tls.Config{
	//	Certificates:       []tls.Certificate{cert},
	//	InsecureSkipVerify: true, // Disable server certificate verification
	//	VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	//		// Implement custom verification logic here
	//		return nil
	//	},
	//}
	addr := "42.192.108.133"
	port := 9200
	url := fmt.Sprintf("http://%s:%d", addr, port)
	EsClient, err := elastic.NewClient(elastic.SetURL(url), elastic.SetSniff(false))
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
	exists, _ := EsClient.Exists().Index("goods").Do(context.Background())
	if !exists {
		EsClient.CreateIndex("goods").BodyString(model.EsGoods{}.GetMapping()).Do(context.Background())
	}
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

//docker run --name elasticsearch -d -e ES_JAVA_OPTS="-Xms512m -Xmx2g" -e "discovery.type=single-node" -e "bootstrap.memory_lock=true" -e "xpack.security.enabled=false" -e "xpack.security.http.ssl.enabled=false" -e "xpack.security.transport.ssl.enabled=false" -p 9200:9200 -p 9300:9300 elasticsearch:8.11.0
//docker run --rm --name jaeger -d -p6831:6831/udp -p16686:16686 -p14268:14268 -e SPAN_STORAGE_TYPE=elasticsearch -e ES_SERVER_URLS="http://42.192.108.133:9200" jaegertracing/all-in-one:latest
//docker run -d --name jaeger --link es:elasticsearch -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 -p 6831:6831/udp -p 6832:6832/udp -p 5778:5778 -p 16686:16686 -p 4317:4317 -p 4318:4318 -p 14250:14250 -p 14268:14268 -p 14269:14269 -p 9411:9411 -e SPAN_STORAGE_TYPE=elasticsearch -e ES_SERVER_URLS="http://127.0.0.1:9200"  jaegertracing/all-in-one:latest
//docker run -d --name elasticsearch -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" docker.elastic.co/elasticsearch/elasticsearch:7.2.1
