package node

import (
	"encoding/json"
	"github.com/godaddy-x/jorm/exception"
	"github.com/godaddy-x/jorm/util"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type HttpNode struct {
	DefaultNode
	Input    *http.Request
	Output   http.ResponseWriter
	TemplDir string
}

type ViewConfig struct {
	BaseDir  string
	Suffix   string
	FileName []string
	Templ    *template.Template
}

func (self *HttpNode) GetHeader(input interface{}) error {
	if self.OverrideFunc.GetHeaderFunc == nil {
		r := input.(*http.Request)
		header := map[string]interface{}{}
		if len(r.Header) > 0 {
			for k, v := range r.Header {
				header[k] = v[0]
			}
		}
		self.Context.Header = header
		return nil
	}
	return self.OverrideFunc.GetHeaderFunc(input)
}

func (self *HttpNode) GetParams(input interface{}) error {
	if self.OverrideFunc.GetParamsFunc == nil {
		r := input.(*http.Request)
		r.ParseForm()
		params := map[string]interface{}{}
		if r.Method == "GET" {
			for k, v := range r.Form {
				params[k] = strings.Join(v, "")
			}
		} else if r.Method == "POST" {
			result, err := ioutil.ReadAll(r.Body)
			if err != nil {
				return err
			}
			r.Body.Close()
			if err := json.Unmarshal(result, &params); err != nil {
				return err
			}
		}
		self.Context.Params = params
		return nil
	}
	return self.OverrideFunc.GetParamsFunc(input)
}

func (self *HttpNode) Html(ctx *Context, view string, data interface{}) error {
	if len(ctx.Response.TemplDir) == 0 {
		return util.Error("templ dir path is nil")
	}
	if len(view) == 0 {
		return util.Error("view file path is nil")
	}
	ctx.Response.ContentEncoding = UTF8
	ctx.Response.ContentType = TEXT_HTML
	ctx.Response.RespView = view
	ctx.Response.RespEntity = data
	return nil
}

func (self *HttpNode) Json(ctx *Context, data interface{}) error {
	if data == nil {
		data = map[string]interface{}{}
	}
	ctx.Response.ContentEncoding = UTF8
	ctx.Response.ContentType = APPLICATION_JSON
	ctx.Response.RespEntity = data
	return nil
}

func (self *HttpNode) Text(ctx *Context, data interface{}) error {
	if data == nil {
		data = map[string]interface{}{}
	}
	ctx.Response.ContentEncoding = UTF8
	ctx.Response.ContentType = TEXT_PLAIN
	ctx.Response.RespEntity = data
	return nil
}

func (self *HttpNode) SetContentType(contentType string) {
	self.Output.Header().Set("Content-Type", contentType)
}

func (self *HttpNode) InitContext(ob, output, input interface{}) error {
	w := output.(http.ResponseWriter)
	r := input.(*http.Request)
	o := ob.(*HttpNode)
	if self.OverrideFunc == nil {
		o.OverrideFunc = &OverrideFunc{}
	}
	if len(self.TemplDir) == 0 {
		if path, err := os.Getwd(); err != nil {
			return err
		} else {
			self.TemplDir = path
		}
	}
	o.OverrideFunc = self.OverrideFunc
	o.SessionManager = self.SessionManager
	o.Output = w
	o.Input = r
	o.Context = &Context{Response: &Response{ContentEncoding: UTF8, ContentType: APPLICATION_JSON, TemplDir: self.TemplDir}}
	if err := o.GetHeader(r); err != nil {
		return err
	}
	if err := o.GetParams(r); err != nil {
		return err
	}
	return nil
}

func (self *HttpNode) ValidSession() error {
	if self.SessionManager == nil {
		return util.Error("session manager is nil")
	}
	sessionId := ""
	if v, b := self.Context.Header[Global.TokenName]; b {
		sessionId = v.(string)
	}
	if len(sessionId) == 0 {
		if v, b := self.Context.Params[Global.TokenName]; b {
			sessionId = v.(string)
		}
	}
	if len(sessionId) == 0 {
		return nil
	}
	session, err := self.SessionManager.ReadSession(sessionId)
	if err != nil {
		return util.Error("read session[", sessionId, "] failure")
	}
	if session == nil {
		return nil
	}
	if !session.IsValid() {
		return util.Error("session[", sessionId, "] invalided")
	}
	self.Context.Session = session
	return nil
}

func (self *HttpNode) PreHandle(handle func(ctx *Context) error) error {
	if handle == nil {
		return nil
	}
	return handle(self.Context)
}

func (self *HttpNode) PostHandle(handle func(resp *Response, err error) error, err error) error {
	if handle != nil {
		if err := handle(self.Context.Response, err); err != nil {
			return err
		}
	}
	return self.Render()
}

func (self *HttpNode) AfterCompletion(handle func(ctx *Context, resp *Response, err error) error, err error) error {
	if handle != nil {
		if err := handle(self.Context, self.Context.Response, err); err != nil {
			return err
		}
	}
	return nil
}

func (self *HttpNode) RenderError(err error) {
	if self.OverrideFunc.RenderErrorFunc == nil {
		out := ex.Catch(err)
		if result, err := json.Marshal(map[string]string{"msg": out.Msg}); err != nil {
			self.Output.WriteHeader(500)
			self.Output.Write([]byte("系统异常"))
		} else {
			self.Output.WriteHeader(out.Code)
			self.Output.Write(result)
		}
	} else {
		self.OverrideFunc.RenderErrorFunc(err)
	}
}

func (self *HttpNode) Render() error {
	switch self.Context.Response.ContentType {
	case TEXT_HTML:
		if templ, err := template.ParseFiles(self.Context.Response.TemplDir + self.Context.Response.RespView); err != nil {
			return err
		} else if err := templ.Execute(self.Output, self.Context.Response.RespEntity); err != nil {
			return err
		}
	case TEXT_PLAIN:
		if result, err := json.Marshal(self.Context.Response.RespEntity); err != nil {
			return err
		} else {
			self.SetContentType(TEXT_PLAIN)
			self.Output.Write(result)
		}
	case APPLICATION_JSON:
		if result, err := json.Marshal(self.Context.Response.RespEntity); err != nil {
			return err
		} else {
			self.SetContentType(APPLICATION_JSON)
			self.Output.Write(result)
		}
	default:
		return ex.Try{Code: 400, Msg: "无效的响应格式"}
	}
	return nil
}

func (self *HttpNode) StartServer() {
	if err := http.ListenAndServe(self.Context.Host, nil); err != nil {
		panic(err)
	}
}

func (self *HttpNode) Proxy(output, input interface{}, handle func(ctx *Context) error) {
	// 1.初始化请求上下文
	ob := &HttpNode{}
	if err := self.InitContext(ob, output, input); err != nil {
		ob.RenderError(ex.Try{400, "请求无效", err, nil})
		return
	}
	// 2.校验会话有效性
	if err := ob.ValidSession(); err != nil {
		ob.RenderError(ex.Try{401, "会话校验失败或已失效", err, nil})
		return
	}
	// 3.上下文前置检测方法
	if err := ob.PreHandle(ob.OverrideFunc.PreHandleFunc); err != nil {
		ob.RenderError(err)
		return
	}
	// 4.执行业务方法成功 -> posthandle视图控制
	result := ob.PostHandle(ob.OverrideFunc.PostHandleFunc, handle(ob.Context))
	// 5.执行afterCompletion方法(传递error参数)
	if err := ob.AfterCompletion(ob.OverrideFunc.AfterCompletionFunc, result); err != nil {
		ob.RenderError(err)
		return
	}
}

func (self *HttpNode) BindFuncByRouter(pattern string, handle func(ctx *Context) error) {
	http.DefaultServeMux.HandleFunc(pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		self.Proxy(w, r, handle)
	}))
}
