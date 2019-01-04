package webapp

import (
	"github.com/godaddy-x/jorm/web"
)

type MyNode struct {
	node.HttpNode
}

func (self *MyNode) test() error {
	return self.Html(map[string]interface{}{"tewt": 1})
}

func Run() {
	self := &MyNode{}
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
	self.CallFunc = callfunc
	self.BindFuncByRouter(self.test, "/test", "/web/index.html")
}
