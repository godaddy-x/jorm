package cache

import "github.com/godaddy-x/jorm/util"

// 缓存管理器
type CacheManager struct {
}

/********************************** 缓存接口定义 **********************************/

// orm数据库接口
type ICache interface {
	// 查询
	Get(key string, input interface{}) (bool, error)
	// 保存/过期时间(秒)
	Put(key string, input interface{}, expire ...int) error
	// 删除
	Del(input ...string) error
	// 查询全部key数量
	Size(pattern ...string) (int, error)
	// 查询全部key
	Keys(pattern ...string) ([]string, error)
	// 查询全部key
	Values(pattern ...string) ([]interface{}, error)
	// 清空全部key-value
	Flush() error
	// 查询队列数据
	Brpop(key string, expire int64, result interface{}) error
	// 发送队列数据
	Rpush(key string, val interface{}) error
}

func (self *CacheManager) Get(key string, input interface{}) (bool, error) {
	return false, util.Error("No implementation method [Get] was found")
}

func (self *CacheManager) Put(key string, input interface{}, expire ...int) error {
	return util.Error("No implementation method [Put] was found")
}

func (self *CacheManager) Del(key ...string) error {
	return util.Error("No implementation method [Del] was found")
}

func (self *CacheManager) Size(pattern ...string) (int, error) {
	return 0, util.Error("No implementation method [Size] was found")
}

func (self *CacheManager) Keys(pattern ...string) ([]string, error) {
	return nil, util.Error("No implementation method [Keys] was found")
}

func (self *CacheManager) Values(pattern ...string) ([]interface{}, error) {
	return nil, util.Error("No implementation method [Values] was found")
}

func (self *CacheManager) Flush() error {
	return util.Error("No implementation method [Flush] was found")
}

func (self *CacheManager) Brpop(key string, expire int64, result interface{}) error {
	return util.Error("No implementation method [Brpop] was found")
}

func (self *CacheManager) Rpush(key string, val interface{}) error {
	return util.Error("No implementation method [Rpush] was found")
}
