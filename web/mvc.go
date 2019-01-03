package main

import (
	"errors"
	"fmt"
	"github.com/godaddy-x/jorm/exception"
	"github.com/godaddy-x/jorm/util"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
)

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

func (self *HttpClient) getHeader(input interface{}) (map[string]interface{}, error) {
	r := input.(*http.Request)
	header := map[string]interface{}{}
	if len(r.Header) > 0 {
		for k, v := range r.Header {
			header[k] = v[0]
		}
	}
	return header, nil
}

func (self *HttpClient) getParams(input interface{}) (map[string]interface{}, error) {
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
			return nil, err
		}
		r.Body.Close()
		if err := util.JsonToObject(string(result), &params); err != nil {
			return nil, err
		}
	}
	return params, nil
}

type Context struct {
	header map[string]interface{}
	params map[string]interface{}
}

type Response struct {
	contentEncoding string
	contentType     string
	respEntity      interface{}
	pagePath        string
}

type HttpClient struct {
	context  *Context
	response *Response
	input    *http.Request
	output   http.ResponseWriter
}

func (self *HttpClient) html(page string, data interface{}) error {
	if len(page) == 0 {
		return errors.New("page path is nil")
	}
	self.response = &Response{contentEncoding: UTF8, contentType: TEXT_HTML, pagePath: page, respEntity: data}
	templ, err := template.ParseFiles(self.response.pagePath)
	if err != nil {
		return err
	}
	if err := templ.Execute(self.output, self.response.respEntity); err != nil {
		return err
	}
	return nil
}

func (self *HttpClient) json(data interface{}) error {
	if data == nil {
		data = map[string]interface{}{}
	}
	self.response = &Response{contentEncoding: UTF8, contentType: APPLICATION_JSON, respEntity: data}
	self.output.Header().Set("Content-Type", APPLICATION_JSON)
	result, err := util.ObjectToJson(data)
	if err != nil {
		return err
	}
	self.output.Write([]byte(result))
	return nil
}

func (self *HttpClient) text(data interface{}) error {
	if data == nil {
		data = map[string]interface{}{}
	}
	self.response = &Response{contentEncoding: UTF8, contentType: TEXT_PLAIN, respEntity: data}
	self.output.Header().Set("Content-Type", TEXT_PLAIN)
	result, err := util.ObjectToJson(data)
	if err != nil {
		return err
	}
	self.output.Write([]byte(result))
	return nil
}

func (self *HttpClient) initContext(output interface{}, input interface{}) error {
	w := output.(http.ResponseWriter)
	r := input.(*http.Request)
	context := &Context{}
	if header, err := self.getHeader(r); err != nil {
		return err
	} else {
		context.header = header
	}
	if params, err := self.getParams(r); err != nil {
		return err
	} else {
		context.params = params
	}
	self.output = w
	self.input = r
	self.context = context
	return nil
}

func (self *HttpClient) writeError(err error) {
	out := ex.Catch(err)
	if result, err := util.ObjectToJson(map[string]string{"msg": out.Msg}); err != nil {
		self.output.Header().Set("Content-Type", APPLICATION_JSON)
		self.output.WriteHeader(500)
		self.output.Write([]byte("系统异常"))
	} else {
		self.output.Header().Set("Content-Type", APPLICATION_JSON)
		self.output.WriteHeader(out.Code)
		self.output.Write([]byte(result))
	}
}

func (self *HttpClient) BindFunc(pattern string, handle func() error) {
	http.DefaultServeMux.HandleFunc(pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := self.initContext(w, r); err != nil {
			self.writeError(ex.Try{400, "请求无效", err, nil})
			return
		}
		if err := handle(); err != nil {
			self.writeError(err)
			return
		}
	}))
}

func (self *HttpClient) test() error {
	// return self.html(util.GetPath()+"/web/index.html", nil)
	return errors.New("我特使错误")
}

func main() {
	self := &HttpClient{}
	self.BindFunc("/test", self.test)
	err := http.ListenAndServe("0.0.0.0:8090", nil)
	if err != nil {
		fmt.Println("http listen failed: ", err)
	}
}
