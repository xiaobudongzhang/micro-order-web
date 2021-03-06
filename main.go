package main

import (
	"fmt"
	"github.com/opentracing/opentracing-go"
	tracer "github.com/xiaobudongzhang/micro-plugins/tracer/myjaeger"
	"github.com/xiaobudongzhang/micro-plugins/tracer/myopentracing/stdmicro"
	"net/http"
	"time"

	"github.com/xiaobudongzhang/micro-basic/common"
	"github.com/xiaobudongzhang/micro-order-web/handler"

	"github.com/micro/cli/v2"
	"github.com/micro/go-micro/v2/registry"
	"github.com/micro/go-micro/v2/registry/etcd"
	log "github.com/micro/go-micro/v2/util/log"
	"github.com/micro/go-micro/v2/web"
	"github.com/micro/go-plugins/config/source/grpc/v2"
	basic "github.com/xiaobudongzhang/micro-basic/basic"
	"github.com/xiaobudongzhang/micro-basic/config"
)

var (
	appName = "orders_web"
	cfg     = &appCfg{}
)

type appCfg struct {
	common.AppCfg
}

func main() {
	initCfg()

	micReg := etcd.NewRegistry(registryOptions)


	t, io, err := tracer.NewTracer(cfg.Name, "")
	if err != nil {
		log.Fatal(err)
	}
	defer io.Close()

	opentracing.SetGlobalTracer(t)

	// create new web service
	service := web.NewService(
		web.Name("mu.micro.book.web.order"),
		web.Version("latest"),
		web.RegisterTTL(time.Second*15),
		web.RegisterInterval(time.Second*10),
		web.Registry(micReg),
		web.Address(":8098"),
	)

	// 初始化服务
	if err := service.Init(
		web.Action(
			func(c *cli.Context) {
				// 初始化handler
				handler.Init()
			}),
	); err != nil {
		log.Fatal(err)
	}

	//新建订单接口
	authHandler := http.HandlerFunc(handler.New)


	service.Handle("/order/new", stdmicro.TracerWrapper(handler.AuthWrapper(authHandler)))
	service.HandleFunc("/order/hello", handler.Hello)

	// run service
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}

func registryOptions(ops *registry.Options) {
	etcdCfg := &common.Etcd{}
	err := config.C().App("etcd", etcdCfg)
	if err != nil {

		log.Log(err)
		panic(err)
	}
	ops.Addrs = []string{fmt.Sprintf("%s:%d", etcdCfg.Host, etcdCfg.Port)}
}

func initCfg() {
	source := grpc.NewSource(
		grpc.WithAddress("127.0.0.1:9600"),
		grpc.WithPath("micro"),
	)

	basic.Init(config.WithSource(source))

	err := config.C().App(appName, cfg)
	if err != nil {
		panic(err)
	}

	log.Logf("配置 cfg:%v", cfg)

	return
}
