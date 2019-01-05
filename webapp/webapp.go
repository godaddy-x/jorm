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

func StartNode() *MyNode {
	my := &MyNode{}
	my.Context = &node.Context{
		Host:       "0.0.0.0:8090",
		Connection: node.HTTP,
	}
	my.SessionManager = &node.CacheSessionManager{}
	my.OverrideFunc = &node.OverrideFunc{
		GetHeaderFunc: nil,
		GetParamsFunc: nil,
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
		RenderErrorFunc: nil,
	}
	my.BindFuncByRouter("/test", my.test)
	my.StartServer()
	return my
}
