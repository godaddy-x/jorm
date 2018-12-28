package ex

import (
	"errors"
	"github.com/godaddy-x/jorm/util"
	"log"
	"strings"
)

/**
 * @author shadow
 * @createby 2018.12.13
 */

const (
	BIZ     = 100000 // 普通业务异常
	SYSTEM  = 999998 // 系统级异常
	UNKNOWN = 999999 // 未知异常
)

type exception struct {
	Code int
	Msg  string
}

func Try(code int, msg ...interface{}) error {
	if s, err := util.ObjectToJson(exception{code, util.AddStr(msg...)}); err != nil {
		log.Println(err)
		return errors.New(util.AddStr("异常转换失败: ", err.Error()))
	} else {
		return errors.New(s)
	}
}

func Catch(err error) exception {
	s := err.Error()
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		ex := exception{}
		if err := util.JsonToObject(s, &ex); err != nil {
			return exception{UNKNOWN, util.AddStr("未知异常错误: ", err.Error())}
		}
		return ex
	}
	return exception{UNKNOWN, s}
}
