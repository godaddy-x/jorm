package jwt

import (
	"github.com/godaddy-x/jorm/util"
	"strings"
)

const (
	JWT    = "JWT"
	SHA256 = "SHA256"
)

type Subject struct {
	Header  *Header
	Payload *Payload
}

type Header struct {
	Nod int64  `json:"nod"` // 认证节点
	Typ string `json:"typ"` // 认证类型
	Alg string `json:"alg"` // 算法类型,默认SHA256
}

type Payload struct {
	Sub string            `json:"sub"` // 签发对象
	Iss string            `json:"iss"` // 签发主体
	Iat int64             `json:"iat"` // 签发时间
	Exp int64             `json:"exp"` // 过期时间
	Nbf int64             `json:"nbf"` // 认证此时间之前不能被接收处理
	Jti string            `json:"jti"` // 认证唯一标识
	Ext map[string]string `json:"ext"` // 扩展信息
}

// 生成Token
func (self *Subject) Generate(secret string) (string, error) {
	if len(secret) == 0 {
		return "", util.Error("secret is nil")
	}
	if self.Payload == nil {
		return "", util.Error("payload is nil")
	}
	if len(self.Payload.Sub) == 0 {
		return "", util.Error("payload.sub is nil")
	}
	if len(self.Payload.Iss) == 0 {
		return "", util.Error("payload.iss is nil")
	}
	if self.Payload.Ext == nil {
		self.Payload.Ext = make(map[string]string, 0)
	}
	if self.Header == nil {
		self.Header = &Header{Typ: JWT, Alg: SHA256, Nod: 0}
	} else if len(self.Header.Typ) == 0 {
		return "", util.Error("header.typ is nil")
	} else if len(self.Header.Alg) == 0 {
		return "", util.Error("header.alg is nil")
	}
	self.Payload.Iat = util.Time()
	self.Payload.Exp = self.Payload.Iat + self.Payload.Exp
	self.Payload.Nbf = self.Payload.Iat + self.Payload.Nbf
	self.Payload.Jti = util.MD5(util.GetUUID(self.Header.Nod))
	h_str, err := util.ObjectToJson(self.Header)
	if err != nil {
		return "", err
	}
	p_str, err := util.ObjectToJson(self.Payload)
	if err != nil {
		return "", err
	}
	h_str = util.Base64URLEncode(h_str)
	p_str = util.Base64URLEncode(p_str)
	if len(h_str) == 0 || len(p_str) == 0 {
		return "", err
	}
	return util.AddStr(h_str, ".", p_str, ".", util.SHA256(util.AddStr(h_str, ".", p_str, ".", secret))), nil
}

// 校验Token
func (self *Subject) Valid(input, secret string) error {
	if len(input) == 0 {
		return util.Error("input is nil")
	}
	if len(secret) == 0 {
		return util.Error("secret is nil")
	}
	sp := strings.Split(input, ".")
	if len(sp) != 3 {
		return util.Error("token is nil")
	}
	if len(sp[0]) == 0 || len(sp[1]) == 0 || len(sp[2]) == 0 {
		return util.Error("message is nil")
	}
	if util.SHA256(util.AddStr(sp[0], ".", sp[1], ".", secret)) != sp[2] {
		return util.Error("token invalid")
	}
	h_str := util.Base64URLDecode(sp[0])
	if len(h_str) == 0 {
		return util.Error("header is nil")
	}
	p_str := util.Base64URLDecode(sp[1])
	if len(p_str) == 0 {
		return util.Error("payload is nil")
	}
	header := &Header{}
	if err := util.JsonToObject(h_str, header); err != nil {
		return util.Error("header is error: ", err.Error())
	}
	payload := &Payload{}
	if err := util.JsonToObject(p_str, payload); err != nil {
		return util.Error("payload is error: ", err.Error())
	}
	current := util.Time()
	if payload.Iat > current {
		return util.Error("iat time invalid")
	}
	if payload.Nbf > current { // 设置了nbf值,大于当前时间,则校验无效
		return util.Error("nbf time invalid")
	}
	if payload.Exp < current {
		return util.Error("exp time invalid")
	}
	if len(payload.Sub) == 0 {
		return util.Error("sub invalid")
	}
	self.Header = header
	self.Payload = payload
	return nil
}
