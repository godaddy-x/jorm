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

var (
	Global = GlobalConfig{TokenName: "token"}
)

type GlobalConfig struct {
	TokenName string
}

type WebNode interface {
	// 初始化上下文
	InitContext(ob, output, input interface{}) error
	// 校验会话
	ValidSession() error
	// 获取请求头数据
	GetHeader(input interface{}) error
	// 获取请求参数
	GetParams(input interface{}) error
	// 设置响应头格式
	SetContentType(contentType string)
	// 核心代理方法
	Proxy(output, input interface{}, handle func(ctx *Context) error)
	// 核心绑定路由方法
	BindFuncByRouter(pattern string, handle func(ctx *Context) error)
	// html响应模式
	Html(ctx *Context, view string, data interface{}) error
	// json响应模式
	Json(ctx *Context, data interface{}) error
	// text响应模式
	Text(ctx *Context, data interface{}) error
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
	// 启动服务
	StartServer()
}
type Context struct {
	Host       string
	Connection int
	Header     map[string]interface{}
	Params     map[string]interface{}
	Session    Session
	Response   *Response
}

type Response struct {
	ContentEncoding string
	ContentType     string
	RespEntity      interface{}
	TemplDir        string
	RespView        string
}

type OverrideFunc struct {
	GetHeaderFunc       func(input interface{}) error
	GetParamsFunc       func(input interface{}) error
	PreHandleFunc       func(ctx *Context) error
	PostHandleFunc      func(resp *Response, err error) error
	AfterCompletionFunc func(ctx *Context, resp *Response, err error) error
	RenderErrorFunc     func(err error) error
	SessionHandleFunc   func(ctx *Context) error
}

type DefaultNode struct {
	Context        *Context
	SessionManager SessionManager
	OverrideFunc   *OverrideFunc
}
