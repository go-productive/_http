package example

import (
	"context"
	"github.com/go-productive/_http"
	pkg1 "github.com/go-productive/_http/example/pkg1"
	"github.com/go-productive/_http/example/pkg2"
	"net/http"
)

type (
	Example struct {
	}
)

// Hello
//
// @RequestMapping{"method":"GET","path":"/hello"}
func (*Example) Hello(req *pkg1.HelloRequest, ctx *_http.DefaultContext) (*pkg2.HelloResponse, error) {
	return &pkg2.HelloResponse{Msg: "hi"}, _http.NewErr(http.StatusInternalServerError, context.DeadlineExceeded)
}

// Hi
//
// @RequestMapping{"method":"GET","path":"/hi"}
func (*Example) Hi(req *pkg2.HelloResponse, ctx *_http.DefaultContext) (*pkg1.HelloRequest, error) {
	return &pkg1.HelloRequest{Msg: "hi"}, _http.NewErr(http.StatusInternalServerError, context.DeadlineExceeded)
}
