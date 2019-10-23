package sqld

import (
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/godaddy-x/jorm/cache"
	"github.com/godaddy-x/jorm/util"
	"time"
)

// mysql配置参数
type MysqlConfig struct {
	DBConfig
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime int
}

// mysql连接管理器
type MysqlManager struct {
	RDBManager
}

func (self *MysqlManager) Get(option ...Option) (*MysqlManager, error) {
	if option != nil && len(option) > 0 {
		if err := self.GetDB(option[0]); err != nil {
			return nil, err
		}
		return self, nil
	} else {
		if err := self.GetDB(); err != nil {
			return nil, err
		}
		return self, nil
	}
}

func (self *MysqlManager) InitConfig(input ...MysqlConfig) error {
	return self.buildByConfig(nil, input...)
}

func (self *MysqlManager) InitConfigAndCache(manager cache.ICache, input ...MysqlConfig) error {
	return self.buildByConfig(manager, input...)
}

func (self *MysqlManager) buildByConfig(manager cache.ICache, input ...MysqlConfig) error {
	for _, conf := range input {
		link := util.AddStr(conf.Username, ":", conf.Password, "@tcp(", conf.Host, ":", util.AnyToStr(conf.Port), ")/"+conf.Database, "?charset=utf8")
		db, err := sql.Open("mysql", link)
		if err != nil {
			panic(util.AddStr("mysql初始化失败: ", err.Error()))
		}
		db.SetMaxIdleConns(conf.MaxIdleConns)
		db.SetMaxOpenConns(conf.MaxOpenConns)
		db.SetConnMaxLifetime(time.Second * time.Duration(conf.ConnMaxLifetime))
		rdb := &RDBManager{}
		rdb.Db = db
		rdb.SlowQuery = conf.SlowQuery
		rdb.SlowLogPath = conf.SlowLogPath
		rdb.Debug = conf.Debug
		rdb.CacheSync = conf.CacheSync
		rdb.CacheManager = manager
		if len(conf.DsName) == 0 {
			rdb.DsName = MASTER
		} else {
			rdb.DsName = conf.DsName
		}
		rdb.initSlowLog()
		rdbs[rdb.DsName] = rdb
	}
	if len(rdbs) == 0 {
		panic("mysql连接初始化失败: 数据源为0")
	}
	return nil
}
