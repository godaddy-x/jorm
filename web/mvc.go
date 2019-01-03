package main

import (
	"encoding/json"
	"fmt"
	"github.com/godaddy-x/jorm/util"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
)

var myTemplate *template.Template

func getHeader(r *http.Request) (map[string]interface{}, error) {
	return nil, nil
}

func getParams(r *http.Request) (map[string]interface{}, error) {
	r.ParseForm()
	params := map[string]interface{}{}
	if r.Method == "GET" {
		fmt.Println("method:", r.Method)
		fmt.Println("username", r.Form["username"])
		fmt.Println("password", r.Form["password"])
		for k, v := range r.Form {
			params[k] = strings.Join(v, "")
		}
	} else if r.Method == "POST" {
		result, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		fmt.Printf("%s\n", result)
		json.Unmarshal(result, &params)
		//m := f.(map[string]interface{})
		//for k, v := range m {
		//	switch vv := v.(type) {
		//	case string:
		//		fmt.Println(k, "is string", vv)
		//	case int:
		//		fmt.Println(k, "is int", vv)
		//	case float64:
		//		fmt.Println(k, "is float64", vv)
		//	case []interface{}:
		//		fmt.Println(k, "is an array:")
		//		for i, u := range vv {
		//			fmt.Println(i, u)
		//		}
		//	default:
		//		fmt.Println(k, "is of a type I don't know how to handle")
		//	}
		//}
		//var s Serverslice;
		//json.Unmarshal([]byte(result), &s)
		//fmt.Println(s.ServersID);
		//for i := 0; i < len(s.Servers); i++ {
		//	fmt.Println(s.Servers[i].ServerName)
		//	fmt.Println(s.Servers[i].ServerIP)
		//}
	}
	return nil, nil
}

type Context struct {
	header   interface{}
	request  interface{}
	response interface{}
	error    error
}

type HttpClient struct {
	context  *Context
	response http.ResponseWriter
	request  *http.Request
}

func initTemplate(fileName string) (err error) {
	myTemplate, err = template.ParseFiles(fileName)
	if err != nil {
		fmt.Println("parse file err:", err)
		return
	}
	return
}

func (self *HttpClient) test() {
	myTemplate.Execute(self.response, nil)
}

func (self *HttpClient) BindFunc(p string, call func()) {
	http.DefaultServeMux.HandleFunc(p, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		self.request = r
		self.response = w
		self.context = &Context{}
		self.test()
	}))
}

func main() {
	initTemplate(util.GetPath() + "/web/index.html")
	self := &HttpClient{}
	self.BindFunc("/test", self.test)
	err := http.ListenAndServe("0.0.0.0:8090", nil)
	if err != nil {
		fmt.Println("http listen failed: ", err)
	}
}
