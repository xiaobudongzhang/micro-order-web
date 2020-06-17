package handler

import (
	"context"
	"encoding/json"
	context2 "github.com/xiaobudongzhang/seata-golang/client/context"
	"github.com/xiaobudongzhang/seata-golang/client/tm"
	"github.com/xiaobudongzhang/seata-golang/client/config"
	"github.com/micro/go-micro/v2/metadata"
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
	clients "github.com/xiaobudongzhang/seata-golang/client"
	//"github.com/xiaobudongzhang/seata-golang/client/tcc"
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
var myid int64
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


	config.InitConf("D:\\micro\\micro-order-web\\conf\\seate_client.yml")
	clients.NewRpcClient()
	tm.Implement(ProxySvc)

	err3 := ProxySvc.CreateSo(w,r)
	response := map[string]interface{}{}
	if err3 != nil {

		response["success"] = false
		response["error"] = Error{
			Detail: err3.Error(),
		}
	}

	response["ref"] = time.Now().UnixNano()


	response["success"] = true
	response["orderId"] = myid


	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func Hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello order web"))
}







type Svc struct {

}

func (svc *Svc) CreateSo(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()

	rootContext := &context2.RootContext{}

	userId := session.GetSession(w, r).Values["userId"].(int64)

	r.ParseForm()

	bookId, _ := strconv.ParseInt(r.Form.Get("bookId"), 10, 10)


	//设置header
	md := make(map[string]string)
	md["xid"] = rootContext.GetXID()
	ctx = metadata.NewContext(ctx, md)


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
		return err1
	}
	if err != nil {
		return err
	}

	myid = rsp.Order.Id
	return  nil
}

var service = &Svc{}

type ProxyService struct {
	*Svc
	CreateSo func(w http.ResponseWriter, r *http.Request) error
}

var methodTransactionInfo = make(map[string]*tm.TransactionInfo)

func init() {
	methodTransactionInfo["CreateSo"] = &tm.TransactionInfo{
		TimeOut:     60000000,
		Name:        "CreateSo",
		Propagation: tm.REQUIRED,
	}
}

func (svc *ProxyService) GetProxyService() interface{} {
	return svc.Svc
}

func (svc *ProxyService) GetMethodTransactionInfo(methodName string) *tm.TransactionInfo {
	return methodTransactionInfo[methodName]
}

var ProxySvc = &ProxyService{
	Svc: service,
}