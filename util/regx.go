package util

import (
	"regexp"
)

const (
	Mobile   = `^1[3|4|5|6|7|8|9][0-9]{9}$`
	IPV4     = `^(25[0-5]|2[0-4]\d|[0-1]?\d?\d)(\.(25[0-5]|2[0-4]\d|[0-1]?\d?\d)){3}$`
	INTEGER  = `(^[1-9]*[1-9][0-9]*$)|0$`
	FLOAT    = `(^[1-9]+(.[0-9]+)?$)|(^[0]+(.[0-9]+)?$)`
	EMAIL    = `^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$`
	ACCOUNT  = `^[a-zA-Z][a-zA-Z0-9_]{5,14}$`
	PASSWORD = `^.{6,18}?`
	URL      = `^((ht|f)tps?):\\/\\/[\\w\\-]+(\\.[\\w\\-]+)+([\\w\\-\\.,@?^=%&:\\/~\\+#]*[\\w\\-\\@?^=%&\\/~\\+#])?$`
)

func IsMobil(s string) bool {
	return ValidPattern(s, Mobile)
}

func IsIPV4(s string) bool {
	return ValidPattern(s, IPV4)
}

func IsInt(s string) bool {
	return ValidPattern(s, INTEGER)
}

func IsFloat(s string) bool {
	return ValidPattern(s, FLOAT)
}

func IsEmail(s string) bool {
	return ValidPattern(s, EMAIL)
}

func IsAccount(s string) bool {
	return ValidPattern(s, ACCOUNT)
}

func IsPassword(s string) bool {
	return ValidPattern(s, PASSWORD)
}

func ValidPattern(content, pattern string) bool {
	cmp := regexp.MustCompile(pattern)
	result := cmp.FindAllString(content, -1)
	if len(result) > 0 {
		return true
	}
	return false
}
