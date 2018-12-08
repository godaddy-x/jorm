package cache

import (
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/godaddy-x/jorm/util"
	"time"
)

var unlockScript = redis.NewScript(1, `
	if redis.call("get", KEYS[1]) == ARGV[1]
	then
		return redis.call("del", KEYS[1])
	else
		return 0
	end
`)

// Lock represents a held lock.
type Lock struct {
	resource string
	token    string
	conn     redis.Conn
	timeout  time.Duration
}

func (lock *Lock) tryLock() (ok bool, err error) {
	status, err := redis.String(lock.conn.Do("SET", lock.key(), lock.token, "EX", int64(lock.timeout/time.Second), "NX"))
	if err == redis.ErrNil {
		// The lock was not successful, it already exists.
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return status == "OK", nil
}

func (lock *Lock) unlock() (err error) {
	_, err = unlockScript.Do(lock.conn, lock.key(), lock.token)
	if lock != nil && lock.conn != nil {
		lock.conn.Close()
	}
	return
}

func (lock *Lock) key() string {
	return fmt.Sprintf("redislock:%s", lock.resource)
}

func (self *RedisManager) getLock(conn redis.Conn, resource string) (lock *Lock, ok bool, err error) {
	timeout := time.Duration(self.LockTimeout) * time.Second
	return self.getLockkWithTimeout(conn, resource, timeout)
}

func (self *RedisManager) getLockkWithTimeout(conn redis.Conn, resource string, timeout time.Duration) (lock *Lock, ok bool, err error) {
	lock = &Lock{resource, util.GetUUID(), conn, timeout}
	ok, err = lock.tryLock()
	if !ok || err != nil {
		conn.Close()
		lock = nil
	}
	return
}

func (self *RedisManager) TryLock(resource string, call func() error) error {
	return self.TryLockWithTimeout(resource, self.LockTimeout, call)
}

func (self *RedisManager) TryLockWithTimeout(resource string, timeout int, call func() error) error {
	client := self.Pool.Get()
	lock, ok, err := self.getLockkWithTimeout(client, resource, time.Duration(timeout)*time.Second)
	if err != nil {
		return errors.New("获取凭证失败: " + err.Error())
	}
	if !ok {
		return errors.New("您的请求[" + resource + "]正在处理,请耐心等待")
	}
	defer lock.unlock()
	return call()
}
