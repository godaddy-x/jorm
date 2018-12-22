package cache

import (
	"github.com/garyburd/redigo/redis"
	"github.com/godaddy-x/jorm/cache"
	"github.com/godaddy-x/jorm/util"
	"time"
)

var (
	MASTER         = "MASTER"
	redis_sessions = make(map[string]*RedisManager, 0)
)

// redis配置参数
type RedisConfig struct {
	DsName      string
	Host        string
	Port        int
	Password    string
	MaxIdle     int
	MaxActive   int
	IdleTimeout int
	Network     string
	LockTimeout int
}

// redis缓存管理器
type RedisManager struct {
	cache.CacheManager
	DsName      string
	LockTimeout int
	Pool        *redis.Pool
}

func (self *RedisManager) InitConfig(input ...RedisConfig) (*RedisManager, error) {
	for e := range input {
		conf := input[e]
		pool := &redis.Pool{MaxIdle: conf.MaxIdle, MaxActive: conf.MaxActive, IdleTimeout: time.Duration(conf.IdleTimeout) * time.Second, Dial: func() (redis.Conn, error) {
			c, err := redis.Dial(conf.Network, util.AddStr(conf.Host, ":", util.AnyToStr(conf.Port)))
			if err != nil {
				return nil, err
			}
			if len(conf.Password) > 0 {
				if _, err := c.Do("AUTH", conf.Password); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		}}
		if len(conf.DsName) > 0 {
			redis_sessions[conf.DsName] = &RedisManager{Pool: pool, DsName: conf.DsName, LockTimeout: conf.LockTimeout}
		} else {
			redis_sessions[MASTER] = &RedisManager{Pool: pool, DsName: MASTER, LockTimeout: conf.LockTimeout}
		}
	}
	if len(redis_sessions) == 0 {
		panic("初始化redis连接池失败: 数据源为0")
	}
	return self, nil
}

func (self *RedisManager) Client(dsname ...string) (*RedisManager, error) {
	var ds string
	if len(dsname) > 0 && len(dsname[0]) > 0 {
		ds = dsname[0]
	} else {
		ds = MASTER
	}
	manager := redis_sessions[ds]
	if manager == nil {
		return nil, util.Error("redis数据源[", ds, "]未找到,请检查...")
	}
	return manager, nil
}

/********************************** redis缓存接口实现 **********************************/

func (self *RedisManager) Get(key string, input interface{}) (bool, error) {
	client := self.Pool.Get()
	defer client.Close()
	value, err := redis.String(client.Do("GET", key))
	if err != nil && err.Error() != "redigo: nil returned" {
		return false, err
	}
	if len(value) > 0 {
		if input == nil {
			return true, nil
		}
		err := util.JsonToObject(value, input);
		return true, err
	}
	return false, nil
}

func (self *RedisManager) Put(key string, input interface{}, expire ...int) error {
	value, err := util.ObjectToJson(input)
	if err != nil {
		return err
	}
	client := self.Pool.Get()
	defer client.Close()
	if len(expire) > 0 && expire[0] > 0 {
		if _, err := client.Do("SET", key, value, "EX", util.AnyToStr(expire[0])); err != nil {
			return err
		}
	} else {
		if _, err := client.Do("SET", key, value); err != nil {
			return err
		}
	}
	return nil
}

func (self *RedisManager) Del(key ...string) error {
	client := self.Pool.Get()
	defer client.Close()
	if len(key) > 0 {
		if _, err := client.Do("DEL", key); err != nil {
			return err
		}
	}
	client.Send("MULTI")
	for e := range key {
		client.Send("DEL", key[e])
	}
	if _, err := client.Do("EXEC"); err != nil {
		return err
	}
	return nil
}

// 数据量大时请慎用
func (self *RedisManager) Keys(pattern ...string) ([]string, error) {
	client := self.Pool.Get()
	defer client.Close()
	p := "*"
	if len(pattern) > 0 && len(pattern[0]) > 0 {
		p = pattern[0]
	}
	keys, err := redis.Strings(client.Do("KEYS", p))
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// 数据量大时请慎用
func (self *RedisManager) Size(pattern ...string) (int, error) {
	keys, err := self.Keys(pattern...)
	if err != nil {
		return 0, err
	}
	return len(keys), nil
}

// 数据量大时请慎用
func (self *RedisManager) Values(pattern ...string) ([]interface{}, error) {
	return nil, util.Error("No implementation method [Values] was found")
}

func (self *RedisManager) Flush() error {
	return util.Error("No implementation method [Flush] was found")
}
