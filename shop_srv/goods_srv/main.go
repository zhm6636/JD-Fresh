package main

import (
	"log"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
	"google.golang.org/grpc"

	"goods_srv/global"

	"goods_srv/logic"
	"goods_srv/proto"
)

func main() {
	// 链路追踪 config
	cfg := config.Configuration{
		ServiceName: "goods_srv",
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
		log.Fatal(err)
	}
	defer closer.Close()

	// 设置全局链路追踪器
	opentracing.SetGlobalTracer(tracer)
	// 创建grpc server
	g := grpc.NewServer(
		// 链路追踪拦截器
		grpc.UnaryInterceptor(otgrpc.OpenTracingServerInterceptor(tracer)),
		grpc.StreamInterceptor(otgrpc.OpenTracingStreamServerInterceptor(tracer)))
	s := &logic.GoodsServer{}
	proto.RegisterGoodsServer(g, s)

	global.InitRPCServer(g)

}
