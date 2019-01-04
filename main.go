package main

import (
	"fmt"
	"github.com/godaddy-x/jorm/webapp"
	"net/http"
)

func main() {
	webapp.Run()
	err := http.ListenAndServe("0.0.0.0:8090", nil)
	if err != nil {
		fmt.Println("http listen failed: ", err)
	}
}
