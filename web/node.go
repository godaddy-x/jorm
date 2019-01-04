package node

const (
	HTTP    = 0
	WEBSOCK = 1
	UTF8    = "UTF-8"
)
const (
	TEXT_HTML        = "text/html"
	TEXT_PLAIN       = "text/plain"
	APPLICATION_JSON = "application/json"
)

type WebNode interface {
	// 初始化上下文
	InitContext(output, input interface{}, view ...interface{}) error
	// 获取请求头数据
	GetHeader(input interface{}) error
	// 获取请求参数
	GetParams(input interface{}) error
	// 设置响应头格式
	SetContentType(contentType string)
	// 核心代理方法
	Proxy(output, input interface{}, handle func() error, view ...interface{})
	// 核心绑定路由方法
	BindFuncByRouter(handle func() error, pattern string, view ...interface{})
	// html响应模式
	Html(data interface{}) error
	// json响应模式
	Json(data interface{}) error
	// text响应模式
	Text(data interface{}) error
	// 前置检测方法(业务方法前执行)
	PreHandle(handle func(ctx *Context) error) error
	// 业务执行方法->自定义处理执行方法(业务方法执行后,视图渲染前执行)
	PostHandle(handle func(resp *Response, err error) error, err error) error
	// 最终响应执行方法(视图渲染后执行,可操作资源释放,保存日志等)
	AfterCompletion(handle func(ctx *Context, resp *Response, err error) error, err error) error
	// 渲染输出
	Render() error
	// 异常错误响应方法
	RenderError(err error)
}
type Context struct {
	Header map[string]interface{}
	Params map[string]interface{}
}

type Response struct {
	ContentEncoding string
	ContentType     string
	RespEntity      interface{}
	RespView        string
}

type CallFunc struct {
	PreHandleFunc       func(ctx *Context) error
	PostHandleFunc      func(resp *Response, err error) error
	AfterCompletionFunc func(ctx *Context, resp *Response, err error) error
	RenderErrorFunc     func(err error) error
	GetHeader           func(input interface{}) error
	GetParams           func(input interface{}) error
}

type DefaultNode struct {
	Context  *Context
	Response *Response
	CallFunc *CallFunc
}
