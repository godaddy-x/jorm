package concurrent

import (
	"errors"
	"sync"
)

type Map struct {
	lock sync.RWMutex
	data map[string]interface{}
}

func NewMap() *Map {
	return &Map{data: map[string]interface{}{}}
}

func (self *Map) Get(key string, callfun func(key string) (interface{}, error)) (interface{}, error) {
	if len(key) == 0 {
		return nil, errors.New("get key is nil")
	}
	if callfun == nil {
		return nil, errors.New("get callfun is nil")
	}
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

func (self *Map) Del(key string, callfun func(key string) (interface{}, error)) error {
	var b bool
	var err error
	self.lock.RLock()
	if _, b = self.data[key]; b {
		self.lock.RUnlock()
		self.lock.Lock()
		if _, b = self.data[key]; b {
			if callfun != nil {
				_, err = callfun(key)
			}
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

func (self *Map) Set(key string, callfun func(key string) (interface{}, error)) (interface{}, error) {
	if len(key) == 0 {
		return nil, errors.New("set key is nil")
	}
	if callfun == nil {
		return nil, errors.New("set callfun is nil")
	}
	self.lock.Lock()
	v, err := callfun(key)
	if v != nil && err == nil {
		self.data[key] = v
	}
	self.lock.Unlock()
	return v, err
}
