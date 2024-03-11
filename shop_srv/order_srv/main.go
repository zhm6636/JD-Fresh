package main

import (
	"fmt"

	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"order_srv/logic"

	"order_srv/global"
	_ "order_srv/global"
	"order_srv/proto"
)

func main() {
	// 链路追踪 config
	cfg := config.Configuration{
		ServiceName: "order_srv",
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans: true,
			//LocalAgentHostPort: "42.192.108.133:6831",
		},
	}

	// 初始化链路追踪器
	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		zap.S().Fatal(err)
	}
	defer closer.Close()

	// 设置全局链路追踪器
	opentracing.SetGlobalTracer(tracer)

	g := grpc.NewServer(
		// 链路追踪拦截器
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)))
	s := &logic.OrderServer{}
	proto.RegisterOrderServer(g, s)

	//启动监听库存归还的延迟队列
	c, _ := rocketmq.NewPushConsumer(
		consumer.WithGroupName(global.Nacos["rocketmq"].(map[string]interface{})["timeoutgroup"].(string)),
		consumer.WithNsResolver(primitive.NewPassthroughResolver([]string{fmt.Sprintf("%s:%d", global.Nacos["rocketmq"].(map[string]interface{})["host"].(string), global.Nacos["rocketmq"].(map[string]interface{})["port"].(int))})),
	)

	if err := c.Subscribe(global.Nacos["rocketmq"].(map[string]interface{})["timeouttopic"].(string), consumer.MessageSelector{}, logic.OrderTimeOut); err != nil {
		fmt.Println("读取消息失败")
	}
	//rocketmq消费者启动
	_ = c.Start()

	global.InitRPCServer(g, c)
}
