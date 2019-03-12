package mc

import (
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/godaddy-x/jorm/cache"
	"github.com/godaddy-x/jorm/util"
	"time"
)

var (
	MASTER      = "MASTER"
	mc_sessions = make(map[string]*MemcacheManager, 0)
)

// memcache配置参数
type MemcacheConfig struct {
	DsName      string
	Host        string
	Port        int
	Password    string
	MaxIdle     int
	MaxActive   int
	IdleTimeout int
}

// redis缓存管理器
type MemcacheManager struct {
	cache.CacheManager
	DsName string
	Pool   *memcache.Client
}

func (self *MemcacheManager) InitConfig(input ...MemcacheConfig) (*MemcacheManager, error) {
	for e := range input {
		conf := input[e]
		mc := memcache.New(util.AddStr(conf.Host, ":", conf.Port))
		mc.Timeout = time.Second * time.Duration(conf.IdleTimeout)
		mc.MaxIdleConns = conf.MaxIdle
		if len(conf.DsName) > 0 {
			mc_sessions[conf.DsName] = &MemcacheManager{Pool: mc, DsName: conf.DsName}
		} else {
			mc_sessions[MASTER] = &MemcacheManager{Pool: mc, DsName: conf.DsName}
		}
	}
	if len(mc_sessions) == 0 {
		panic("初始化memcache连接池失败: 数据源为0")
	}
	return self, nil
}

func (self *MemcacheManager) Client(dsname ...string) (*MemcacheManager, error) {
	var ds string
	if len(dsname) > 0 && len(dsname[0]) > 0 {
		ds = dsname[0]
	} else {
		ds = MASTER
	}
	manager := mc_sessions[ds]
	if manager == nil {
		return nil, util.Error("redis数据源[", ds, "]未找到,请检查...")
	}
	return manager, nil
}

/********************************** memcache缓存接口实现 **********************************/

func (self *MemcacheManager) Get(key string, input interface{}) (bool, error) {
	it, err := self.Pool.Get(key)
	if err != nil {
		return false, err
	}
	if string(it.Key) != key {
		return false, err
	}
	value := string(it.Value)
	if len(value) > 0 {
		if input == nil {
			return true, nil
		}
		err := util.JsonToObject(value, input);
		return true, err
	}
	return false, nil
}

func (self *MemcacheManager) Put(key string, input interface{}, expire ...int) error {
	value, err := util.ObjectToJson(input)
	if err != nil {
		return err
	}
	if len(expire) > 0 && expire[0] > 0 {
		if err := self.Pool.Set(&memcache.Item{Key: key, Value: []byte(value), Expiration: int32(expire[0])}); err != nil {
			return err
		}
	} else {
		if err := self.Pool.Set(&memcache.Item{Key: key, Value: []byte(value)}); err != nil {
			return err
		}
	}
	return nil
}

func (self *MemcacheManager) Del(key ...string) error {
	if len(key) > 0 {
		for e := range key {
			self.Pool.Delete(key[e])
		}
	}
	return nil
}

// 数据量大时请慎用
func (self *MemcacheManager) Keys(pattern ...string) ([]string, error) {
	return nil, nil
}

// 数据量大时请慎用
func (self *MemcacheManager) Size(pattern ...string) (int, error) {
	return 0, nil
}

// 数据量大时请慎用
func (self *MemcacheManager) Values(pattern ...string) ([]interface{}, error) {
	return nil, util.Error("No implementation method [Values] was found")
}

func (self *MemcacheManager) Flush() error {
	return util.Error("No implementation method [Flush] was found")
}
