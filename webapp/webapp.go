package webapp

import (
	"github.com/godaddy-x/jorm/web"
)

type MyNode struct {
	node.HttpNode
}

func (self *MyNode) test(ctx *node.Context) error {
	return self.Html(ctx, "/web/index.html", map[string]interface{}{"tewt": 1})
	//return self.Json(ctx, map[string]interface{}{"tewt": 1})
}

func Run() {
	my := &MyNode{}
	callfunc := &node.CallFunc{
		PreHandleFunc: func(ctx *node.Context) error {
			return nil
		},
		PostHandleFunc: func(resp *node.Response, err error) error {
			resp.RespEntity = map[string]interface{}{"sssss": 3}
			return err
		},
		AfterCompletionFunc: func(ctx *node.Context, resp *node.Response, err error) error {
			return err
		},
	}
	my.CallFunc = callfunc
	my.BindFuncByRouter("/test", my.test)
}
