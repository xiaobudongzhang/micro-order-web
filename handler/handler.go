package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/micro/go-micro/v2/client"
	"github.com/micro/go-micro/v2/util/log"
	auth "github.com/xiaobudongzhang/micro-auth/proto/auth"
	invS "github.com/xiaobudongzhang/micro-inventory-srv/proto/inventory"
	orders "github.com/xiaobudongzhang/micro-order-srv/proto/order"
	"github.com/xiaobudongzhang/micro-plugins/session"
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
	hystrix_go.DefaultErrorPercentThresHold = 1

	serviceClient = orders.NewOrdersService("mu.micro.book.service.order", client.DefaultClient)
	authClient = auth.NewService("mu.micro.book.service.auth", client.DefaultClient)
}

func New(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Logf("非法请求")
		http.Error(w, "非法请求", 400)
		return
	}

	r.ParseForm()

	bookId, _ := strconv.ParseInt(r.Form.Get("bookId"), 10, 10)

	response := map[string]interface{}{}

	rsp, err := serviceClient.New(context.TODO(), &orders.Request{
		BookId: bookId,
		UserId: session.GetSession(w, r).Values["userId"].(int64),
	})

	response["ref"] = time.Now().UnixNano()

	if err != nil {
		response["success"] = false
		response["error"] = Error{
			Detail: err.Error(),
		}
	} else {
		response["success"] = true
		response["orderId"] = rsp.Order.Id
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func Hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello order web"))
}
