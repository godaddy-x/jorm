package jwt

/**
 * @author shadow
 * @createby 2018.12.13
 */

import (
	"github.com/godaddy-x/jorm/util"
	"strings"
)

const (
	JWT          = "JWT"
	SHA256       = "SHA256"
	QUARTER_HOUR = int64(900000);
	HALF_HOUR    = int64(3600000);
	TWO_WEEK     = int64(1209600000);
)

type Subject struct {
	Header  *Header
	Payload *Payload
}

type Authorization struct {
	AccessTime   int64  `json:"accessTime"`   // 授权时间
	AccessToken  string `json:"accessToken"`  // 授权Token
	RefreshToken string `json:"refreshToken"` // 续期Token
}

type Header struct {
	Nod int64  `json:"nod"` // 认证节点
	Typ string `json:"typ"` // 认证类型
	Alg string `json:"alg"` // 算法类型,默认SHA256
}

type Payload struct {
	Sub string            `json:"sub"` // 用户主体
	Aud string            `json:"aud"` // 接收token主体
	Iss string            `json:"iss"` // 签发token主体
	Iat int64             `json:"iat"` // 授权token时间
	Exp int64             `json:"exp"` // 授权token过期时间
	Rxp int64             `json:"rxp"` // 续期token过期时间
	Nbf int64             `json:"nbf"` // 定义在什么时间之前,该token都是不可用的
	Jti string            `json:"jti"` // 唯一身份标识,主要用来作为一次性token,从而回避重放攻击
	Ext map[string]string `json:"ext"` // 扩展信息
}

// 生成Token
func (self *Subject) Generate(secret string, refresh ...bool) (*Authorization, error) {
	if len(secret) == 0 {
		return nil, util.Error("secret is nil")
	}
	if self.Payload == nil {
		return nil, util.Error("payload is nil")
	}
	if len(self.Payload.Sub) == 0 {
		return nil, util.Error("payload.sub is nil")
	}
	if len(self.Payload.Iss) == 0 {
		return nil, util.Error("payload.iss is nil")
	}
	if self.Payload.Ext == nil {
		self.Payload.Ext = make(map[string]string, 0)
	}
	if self.Header == nil {
		self.Header = &Header{Typ: JWT, Alg: SHA256, Nod: 0}
	} else if len(self.Header.Typ) == 0 {
		return nil, util.Error("header.typ is nil")
	} else if len(self.Header.Alg) == 0 {
		return nil, util.Error("header.alg is nil")
	}
	self.Payload.Jti = util.SHA256(util.GetUUID(self.Header.Nod))
	self.Payload.Iat = util.Time()
	if self.Payload.Exp <= 0 {
		self.Payload.Exp = self.Payload.Iat + HALF_HOUR
	} else {
		self.Payload.Exp = self.Payload.Iat + self.Payload.Exp
	}
	if refresh == nil || len(refresh) == 0 || !refresh[0] {
		self.Payload.Nbf = self.Payload.Iat + self.Payload.Nbf
		if self.Payload.Rxp > 0 {
			if self.Payload.Rxp <= 0 || self.Payload.Rxp > TWO_WEEK {
				self.Payload.Rxp = self.Payload.Iat + TWO_WEEK
			} else {
				self.Payload.Rxp = self.Payload.Iat + self.Payload.Rxp
			}
		} else {
			self.Payload.Rxp = self.Payload.Exp
		}
		if self.Payload.Exp > self.Payload.Rxp {
			self.Payload.Exp = self.Payload.Rxp
		}
	}
	if self.Payload.Iat > self.Payload.Exp {
		return nil, util.Error("the exp must be longer than the iat")
	}
	if self.Payload.Iat > self.Payload.Rxp {
		return nil, util.Error("the rxp must be longer than the iat")
	}
	h_str, err := util.ObjectToJson(self.Header)
	if err != nil {
		return nil, err
	}
	p_str, err := util.ObjectToJson(self.Payload)
	if err != nil {
		return nil, err
	}
	h_str = util.Base64URLEncode(h_str)
	p_str = util.Base64URLEncode(p_str)
	if len(h_str) == 0 || len(p_str) == 0 {
		return nil, err
	}
	accessToken := util.AddStr(h_str, ".", p_str, ".", util.SHA256(util.AddStr(h_str, ".", p_str, ".", secret)))
	accessTime := self.Payload.Iat
	refreshToken := util.SHA256(util.AddStr(accessToken, ".", util.AnyToStr(accessTime), ".", secret))
	return &Authorization{AccessToken: accessToken, RefreshToken: refreshToken, AccessTime: accessTime}, nil
}

// 校验Token
func (self *Subject) Valid(accessToken, secret string, aud ...string) error {
	if len(accessToken) == 0 {
		return util.Error("accessToken is nil")
	}
	if len(secret) == 0 {
		return util.Error("secret is nil")
	}
	sp := strings.Split(accessToken, ".")
	if len(sp) != 3 {
		return util.Error("accessToken is nil")
	}
	if len(sp[0]) == 0 || len(sp[1]) == 0 || len(sp[2]) == 0 {
		return util.Error("message is nil")
	}
	if util.SHA256(util.AddStr(sp[0], ".", sp[1], ".", secret)) != sp[2] {
		return util.Error("accessToken invalid")
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
	if len(payload.Aud) > 0 {
		if aud == nil || len(aud) == 0 || aud[0] != payload.Aud {
			return util.Error("aud invalid")
		}
	}
	self.Header = header
	self.Payload = payload
	return nil
}

// 续期Token
func (self *Subject) Refresh(accessToken, refreshToken, secret string, accessTime int64, interval ...int64) (*Authorization, error) {
	current := util.Time()
	if accessTime > current {
		return nil, util.Error("accessTime error")
	}
	if interval == nil || len(interval) == 0 || interval[0] <= 0 {
		if current-accessTime < QUARTER_HOUR {
			return nil, util.Error("it must be more than ", util.AnyToStr(QUARTER_HOUR), " milliseconds")
		}
	} else {
		if current-accessTime < interval[0] {
			return nil, util.Error("it must be more than ", util.AnyToStr(interval[0]), " milliseconds")
		}
	}
	validRefreshToken := util.SHA256(util.AddStr(accessToken, ".", util.AnyToStr(accessTime), ".", secret))
	if validRefreshToken != refreshToken {
		return nil, util.Error("refreshToken invalid")
	}
	if err := self.Valid(accessToken, secret); err != nil {
		return nil, err
	}
	if self.Payload.Iat != accessTime {
		return nil, util.Error("accessTime invalid")
	}
	if self.Payload.Rxp < util.Time() {
		return nil, util.Error("refreshToken expired")
	}
	payload := &Payload{
		Sub: self.Payload.Sub,
		Aud: self.Payload.Aud,
		Iss: self.Payload.Iss,
		Exp: self.Payload.Exp - self.Payload.Iat,
		Rxp: self.Payload.Rxp,
		Nbf: self.Payload.Nbf,
		Ext: self.Payload.Ext,
	}
	subject := &Subject{Header: self.Header, Payload: payload}
	if rs, err := subject.Generate(secret, true); err != nil {
		return nil, err
	} else {
		return rs, nil
	}
}
