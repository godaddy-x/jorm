package log_test

import (
	"errors"
	"fmt"
	"github.com/godaddy-x/jorm/log"
	"github.com/godaddy-x/jorm/util"
	"testing"
)

func TestZap(t *testing.T) {
	file := &log.FileConfig{
		Filename:   "/Users/shadowsick/go/src/github.com/godaddy-x/spikeProxy1.log", // 日志文件路径
		MaxSize:    1,                                                               // 每个日志文件保存的最大尺寸 单位：M
		MaxBackups: 30,                                                              // 日志文件最多保存多少个备份
		MaxAge:     7,                                                               // 文件最多保存多少天
		Compress:   true,                                                            // 是否压缩
	}
	config := &log.ZapConfig{
		Level:      log.DEBUG,
		Console:    true,
		FileConfig: file,
		Callfunc: func(b []byte) error {
			fmt.Println(string(b))
			return nil
		},
	}
	log.InitDefaultLog(config)
	a := errors.New("my")
	b := errors.New("ow")
	c := []error{a, b}
	log.Info("log 初始化成功", 0, log.String("test", "w"), log.Any("wo", map[string]interface{}{"yy": 45}), log.AddError(c...))
	log.Println("test")
	fmt.Println(util.Time2Str(util.Time()))

}
