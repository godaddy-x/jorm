package concurrent

import "sync"

type ConcurrentMap struct {
	lock sync.RWMutex
	data map[string]interface{}
}

func (self *ConcurrentMap) Get(key string, callfun func(key string) (interface{}, error)) (interface{}, error) {
	var b bool
	var v interface{}
	var err error
	self.lock.RLock()
	if v, b = self.data[key]; !b {
		self.lock.RUnlock()
		self.lock.Lock()
		if v, b = self.data[key]; !b {
			v, err = callfun(key)
			if v != nil && err == nil {
				self.data[key] = v
			}
		}
		self.lock.Unlock()
	} else {
		self.lock.RUnlock()
	}
	return v, err
}

func (self *ConcurrentMap) Del(key string, callfun func(key string) (interface{}, error)) error {
	var b bool
	var err error
	self.lock.RLock()
	if _, b = self.data[key]; b {
		self.lock.RUnlock()
		self.lock.Lock()
		if _, b = self.data[key]; b {
			_, err = callfun(key)
			if err == nil {
				delete(self.data, key)
			}
		}
		self.lock.Unlock()
	} else {
		self.lock.RUnlock()
	}
	return err
}

func (self *ConcurrentMap) Set(key string, callfun func(key string) (interface{}, error)) (interface{}, error) {
	self.lock.Lock()
	v, err := callfun(key)
	if v != nil && err == nil {
		self.data[key] = v
	}
	self.lock.Unlock()
	return v, err
}
