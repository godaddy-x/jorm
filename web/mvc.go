package main

import (
	"fmt"
	"github.com/godaddy-x/jorm/util"
	"html/template"
	"net/http"
)

var myTemplate *template.Template

func initTemplate(fileName string) (err error) {
	myTemplate, err = template.ParseFiles(fileName)
	if err != nil {
		fmt.Println("parse file err:", err)
		return
	}
	return
}

func main() {
	initTemplate(util.GetPath() + "/web/index.html")
	http.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		myTemplate.Execute(w, nil)
	})
	err := http.ListenAndServe("0.0.0.0:8090", nil)
	if err != nil {
		fmt.Println("http listen failed: ", err)
	}
}
