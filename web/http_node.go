package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/godaddy-x/jorm/exception"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type HttpNode struct {
	DefaultNode
	Input       *http.Request
	Output      http.ResponseWriter
	ViewPathDir string
}

func (self *HttpNode) GetHeader(input interface{}) error {
	if self.CallFunc.GetHeader == nil {
		r := input.(*http.Request)
		header := map[string]interface{}{}
		if len(r.Header) > 0 {
			for k, v := range r.Header {
				header[k] = v[0]
			}
		}
		return nil
	}
	return self.CallFunc.GetHeader(input)
}

func (self *HttpNode) GetParams(input interface{}) error {
	if self.CallFunc.GetParams == nil {
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
		return nil
	}
	return self.CallFunc.GetParams(input)
}

func (self *HttpNode) Html(data interface{}) error {
	if len(self.ViewPathDir) == 0 {
		return errors.New("view dir path is nil")
	}
	if len(self.Response.RespView) == 0 {
		return errors.New("view file path is nil")
	}
	self.Response.ContentEncoding = UTF8
	self.Response.ContentType = TEXT_HTML
	self.Response.RespEntity = data
	return nil
}

func (self *HttpNode) Json(data interface{}) error {
	if data == nil {
		data = map[string]interface{}{}
	}
	self.Response.ContentEncoding = UTF8
	self.Response.ContentType = APPLICATION_JSON
	self.Response.RespEntity = data
	return nil
}

func (self *HttpNode) Text(data interface{}) error {
	if data == nil {
		data = map[string]interface{}{}
	}
	self.Response.ContentEncoding = UTF8
	self.Response.ContentType = TEXT_PLAIN
	self.Response.RespEntity = data
	return nil
}

func (self *HttpNode) SetContentType(contentType string) {
	self.Output.Header().Set("Content-Type", contentType)
}

func (self *HttpNode) InitContext(output, input interface{}, view ...interface{}) error {
	w := output.(http.ResponseWriter)
	r := input.(*http.Request)
	context := &Context{}
	response := &Response{ContentEncoding: UTF8, ContentType: APPLICATION_JSON}
	if err := self.GetHeader(r); err != nil {
		return err
	}
	if err := self.GetParams(r); err != nil {
		return err
	}
	if len(view) > 0 {
		if v0, b := view[0].(string); b && len(v0) > 0 {
			response.RespView = v0
			response.ContentType = TEXT_HTML
		}
	} else {
		w.Header().Set("Content-Type", APPLICATION_JSON)
	}
	if self.CallFunc == nil {
		self.CallFunc = &CallFunc{}
	}
	if len(self.ViewPathDir) == 0 {
		if path, err := os.Getwd(); err != nil {
			return err
		} else {
			self.ViewPathDir = path
		}
	}
	self.Output = w
	self.Input = r
	self.Context = context
	self.Response = response
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
		if err := handle(self.Response, err); err != nil {
			return err
		}
	}
	return self.Render()
}

func (self *HttpNode) AfterCompletion(handle func(ctx *Context, resp *Response, err error) error, err error) error {
	if handle != nil {
		if err := handle(self.Context, self.Response, err); err != nil {
			return err
		}
	}
	return nil
}

func (self *HttpNode) RenderError(err error) {
	if self.CallFunc.RenderErrorFunc == nil {
		out := ex.Catch(err)
		if result, err := json.Marshal(map[string]string{"msg": out.Msg}); err != nil {
			self.Output.WriteHeader(500)
			self.Output.Write([]byte("系统异常"))
		} else {
			self.Output.WriteHeader(out.Code)
			self.Output.Write(result)
		}
	} else {
		self.CallFunc.RenderErrorFunc(err)
	}
}

func (self *HttpNode) Render() error {
	switch self.Response.ContentType {
	case TEXT_HTML:
		if templ, err := template.ParseFiles(self.ViewPathDir + self.Response.RespView); err != nil {
			return err
		} else if err := templ.Execute(self.Output, self.Response.RespEntity); err != nil {
			return err
		}
	case TEXT_PLAIN:
		self.SetContentType(TEXT_PLAIN)
		if result, err := json.Marshal(self.Response.RespEntity); err != nil {
			return err
		} else {
			fmt.Fprint(self.Output, result)
		}
	case APPLICATION_JSON:
		if result, err := json.Marshal(self.Response.RespEntity); err != nil {
			return err
		} else {
			self.Output.Write(result)
		}
	default:
		return errors.New("无效的响应格式")
	}
	return nil
}

func (self *HttpNode) Proxy(output, input interface{}, handle func() error, view ...interface{}) {
	// 1.初始化请求上下文
	if err := self.InitContext(output, input, view...); err != nil {
		self.RenderError(ex.Try{400, "请求无效", err, nil})
		return
	}
	// 2.上下文前置检测方法
	if err := self.PreHandle(self.CallFunc.PreHandleFunc); err != nil {
		self.RenderError(err)
		return
	}
	// 3.执行业务方法成功 -> posthandle视图控制
	result := self.PostHandle(self.CallFunc.PostHandleFunc, handle())
	// 4.执行afterCompletion方法(传递error参数)
	if err := self.AfterCompletion(self.CallFunc.AfterCompletionFunc, result); err != nil {
		self.RenderError(err)
		return
	}
}

func (self *HttpNode) BindFuncByRouter(handle func() error, pattern string, view ...interface{}) {
	http.DefaultServeMux.HandleFunc(pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		self.Proxy(w, r, handle, view...)
	}))
}
