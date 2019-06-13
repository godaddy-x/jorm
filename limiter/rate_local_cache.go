package rate

import (
	"github.com/godaddy-x/freego/cache"
	"sync"
)

type RateLimiter struct {
	mu    sync.Mutex
	cache cache.ICache
}

func NewLocalLimiter(c cache.ICache) *RateLimiter {
	if c == nil {
		return &RateLimiter{cache: new(cache.LocalMapManager).NewCache(30, 3)}
	}
	return &RateLimiter{cache: c}
}

// key=过滤关键词 limit=速率 bucket=容量 expire=过期时间/秒
func (self *RateLimiter) getLimiter(key string, limit Limit, bucket int, expire int) *Limiter {
	self.mu.Lock()
	defer self.mu.Unlock()
	var limiter *Limiter
	if v, b, _ := self.cache.Get(key, nil); b {
		limiter = v.(*Limiter)
	} else {
		limiter = NewLimiter(limit, bucket)
	}
	return self.setLimiter(key, limiter, expire)
}

func (self *RateLimiter) setLimiter(key string, limiter *Limiter, expire int) *Limiter {
	self.cache.Put(key, limiter, expire)
	return limiter
}

// return false=接受请求 true=拒绝请求
func (self *RateLimiter) Validate(key string, limit Limit, bucket int, expire int) bool {
	return !self.getLimiter(key, limit, bucket, expire).Allow()
}
