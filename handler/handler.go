package handler

import (
	"context"
	"encoding/json"

	"net/http"
	"strconv"
	"time"

	hystrix_go "github.com/afex/hystrix-go/hystrix"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/util/log"
	"github.com/micro/go-plugins/wrapper/breaker/hystrix/v2"
	auth "github.com/xiaobudongzhang/micro-auth/proto/auth"
	invS "github.com/xiaobudongzhang/micro-inventory-srv/proto/inventory"
	orders "github.com/xiaobudongzhang/micro-order-srv/proto/order"
	"github.com/xiaobudongzhang/micro-plugins/session"
	//context2 "github.com/xiaobudongzhang/seata-golang/client/context"
	//"github.com/xiaobudongzhang/seata-golang/client/tm"
)

var (
	serviceClient orders.OrdersService
	authClient    auth.Service
	invClient     invS.InventoryService
)

type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func Init() {
	hystrix_go.DefaultVolumeThreshold = 1
	hystrix_go.DefaultErrorPercentThreshold = 1

	c1 := hystrix.NewClientWrapper()(client.DefaultClient)

	c1.Init(
		client.Retries(3),
		client.Retry(
			func(ctx context.Context, req client.Request, retryCount int, err error) (bool, error) {
				log.Log(req.Method(), retryCount, "client retry")
				return true, nil
			}),
	)

	serviceClient = orders.NewOrdersService("mu.micro.book.service.order", c1)
	authClient = auth.NewService("mu.micro.book.service.auth", c1)
	invClient = invS.NewInventoryService("mu.micro.book.service.inventory", c1)
}

func New(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Logf("非法请求")
		http.Error(w, "非法请求", 400)
		return
	}
	ctx := r.Context()


	userId := session.GetSession(w, r).Values["userId"].(int64)

	r.ParseForm()

	bookId, _ := strconv.ParseInt(r.Form.Get("bookId"), 10, 10)
	response := map[string]interface{}{}



	//rootContext := ctx.(*context2.RootContext)
	//businessActionContextA := &context2.BusinessActionContext{
	//	RootContext: rootContext,
	//	ActionContext: make(map[string]interface{}),
	//}
	//
	//
	//businessActionContextB := &context2.BusinessActionContext{
	//	RootContext: rootContext,
	//	ActionContext: make(map[string]interface{}),
	//}
	//businessActionContextB.ActionContext["hello"] = "hello world,this is from BusinessActionContext B"
	//


	rsp1, err1 := invClient.Sell(ctx, &invS.Request{
		BookId: bookId, UserId: userId,
	})

	rsp, err := serviceClient.New(ctx, &orders.Request{
		BookId: bookId,
		UserId: userId,
		OrderId:rsp1.InvH.Id,
	})

	if err1 != nil {
		log.Logf("sell 调用库存服务失败：%s", err1.Error())
		return
	}
	if err != nil {
		response["success"] = false
		response["error"] = Error{
			Detail: err.Error(),
		}
	}






	response["ref"] = time.Now().UnixNano()


	response["success"] = true
	response["orderId"] = rsp.Order.Id


	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func Hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello order web"))
}
