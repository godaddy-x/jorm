package node

import (
	"errors"
	"github.com/godaddy-x/jorm/jwt"
	"github.com/godaddy-x/jorm/util"
)

type Session interface {
	GetId() string

	GetStartTimestamp() int64

	GetLastAccessTime() int64

	GetTimeout() (int64, error) // 抛出校验会话异常

	SetTimeout(t int64) error // 抛出校验会话异常

	GetHost() string

	Touch() error // 刷新最后授权时间,抛出校验会话异常

	Stop() error // 抛出校验会话异常

	GetAttributeKeys() ([]string, error) // 抛出校验会话异常

	GetAttribute(k string) (interface{}, error) // 抛出校验会话异常

	SetAttribute(k string, v interface{}) error // 抛出校验会话异常

	RemoveAttribute(k string) error // 抛出校验会话异常

	Validate() error

	IsValid() bool
}

type JWTSession struct {
	Id             string
	StartTimestamp int64
	LastAccessTime int64
	Timeout        int64
	StopTime       int64
	Host           string
	IsExpire       bool
	KV             map[string]interface{}
}

func (self *JWTSession) Build(sub *jwt.Subject, srt string) (*jwt.Authorization, error) {
	author, err := sub.Generate(srt);
	if err != nil {
		return nil, err
	}
	self.Id = author.Signature
	self.Host = sub.Payload.Aud
	self.Timeout = sub.Payload.Exp
	self.StartTimestamp = author.AccessTime
	self.LastAccessTime = author.AccessTime
	self.KV = map[string]interface{}{}
	return author, nil
}

func (self *JWTSession) GetId() string {
	return self.Id
}

func (self *JWTSession) GetStartTimestamp() int64 {
	return self.StartTimestamp
}

func (self *JWTSession) GetLastAccessTime() int64 {
	return self.LastAccessTime
}

func (self *JWTSession) GetTimeout() (int64, error) {
	if err := self.Validate(); err != nil {
		return 0, err
	}
	return self.Timeout, nil
}

func (self *JWTSession) SetTimeout(t int64) error {
	if err := self.Validate(); err != nil {
		return err
	}
	self.Timeout = t
	return nil
}

func (self *JWTSession) GetHost() string {
	return self.Host
}

func (self *JWTSession) Touch() error {
	if err := self.Validate(); err != nil {
		return err
	}
	self.LastAccessTime = util.Time()
	return nil
}

func (self *JWTSession) Stop() error {
	if err := self.Validate(); err != nil {
		return err
	}
	self.StopTime = util.Time()
	return nil
}

func (self *JWTSession) Expire() error {
	if err := self.Validate(); err != nil {
		return err
	}
	self.Stop()
	self.IsExpire = true
	return nil
}

func (self *JWTSession) Validate() error {
	if self.IsExpire {
		return errors.New("session expired")
	} else if util.Time() > (self.LastAccessTime + self.Timeout) {
		self.Expire()
		return errors.New("session timeout")
	}
	return nil
}

func (self *JWTSession) IsValid() bool {
	if !self.IsExpire && self.StopTime == 0 {
		return true
	}
	return false
}

func (self *JWTSession) GetAttributeKeys() ([]string, error) {
	if err := self.Validate(); err != nil {
		return nil, err
	}
	keys := []string{}
	for k, _ := range self.KV {
		keys = append(keys, k)
	}
	return keys, nil
}

func (self *JWTSession) GetAttribute(k string) (interface{}, error) {
	if err := self.Validate(); err != nil {
		return nil, err
	}
	if len(k) == 0 {
		return nil, nil
	}
	if v, b := self.KV[k]; b {
		return v, nil
	}
	return nil, nil
}

func (self *JWTSession) SetAttribute(k string, v interface{}) error {
	if err := self.Validate(); err != nil {
		return err
	}
	self.KV[k] = v
	return nil
}

func (self *JWTSession) RemoveAttribute(k string) error {
	if len(k) == 0 {
		return nil
	}
	if _, b := self.KV[k]; b {
		delete(self.KV, k)
	}
	return nil
}

type SessionManager interface {
	Create(s Session) error // 保存session

	ReadSession(s string) (Session, error) // 通过id读取session,抛出未知session异常

	Update(s Session) error // 更新session,抛出未知session异常

	Delete(s Session) error // 删除session,抛出未知session异常

	GetActiveSessions() []Session // 获取活动的session集合
}

type CacheSessionManager struct {
}

func (self *CacheSessionManager) Create(s Session) error {
	if len(s.GetId()) == 0 {
		return errors.New("sessionId is nill")
	}
	return nil
}

func (self *CacheSessionManager) ReadSession(s string) (Session, error) {
	return nil, nil
}

func (self *CacheSessionManager) Update(s Session) error {
	return nil
}

func (self *CacheSessionManager) Delete(s Session) error {
	return nil
}

func (self *CacheSessionManager) GetActiveSessions() []Session {
	return nil
}
