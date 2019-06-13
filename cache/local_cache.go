package cache

import (
	"time"
)

// 本地缓存管理器
type LocalMapManager struct {
	CacheManager
	c *Cache
}

// a默认缓存时间/分钟 b默认校验数据间隔时间/分钟
func NewLocalCache(a, b int) *LocalMapManager {
	c := New(time.Duration(a)*time.Minute, time.Duration(b)*time.Minute)
	return &LocalMapManager{c: c}
}

func (self *LocalMapManager) GetBy(key string, input interface{}) (interface{}, bool, error) {
	v, b := self.c.Get(key)
	if v != nil {
		input = v
	}
	return input, b, nil
}

func (self *LocalMapManager) Get(key string, input interface{}) (bool, error) {
	v, b := self.c.Get(key)
	if v != nil {
		input = v
	}
	return b, nil
}

func (self *LocalMapManager) Put(key string, input interface{}, expire ...int) error {
	if expire != nil && len(expire) > 0 {
		self.c.Set(key, input, time.Duration(expire[0])*time.Second)
	} else {
		self.c.SetDefault(key, input)
	}
	return nil
}

func (self *LocalMapManager) Del(key ...string) error {
	if key != nil {
		for _, v := range key {
			self.c.Delete(v)
		}
	}
	return nil
}

// 数据量大时请慎用
func (self *LocalMapManager) Size(pattern ...string) (int, error) {
	return self.c.ItemCount(), nil
}

func (self *LocalMapManager) Flush() error {
	self.c.Flush()
	return nil
}
