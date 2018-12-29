package gauth

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	"github.com/godaddy-x/jorm/util"
	"time"
)

const (
	windowSize  = 3
	secretSize  = 10
	base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	// seed        = "g8GjEvTbW5oVSV7avL47357438reyhreyuryetredLDVKs2m0QN7vxRs2im5MDaNCWGmcD2rvcZx"
)

//transform int64 to []byte
func toBytes(value int64) []byte {
	var result []byte
	mask := int64(0xFF)
	shifts := [8]uint16{56, 48, 40, 32, 24, 16, 8, 0}
	for _, shift := range shifts {
		result = append(result, byte((value>>shift)&mask))
	}
	return result
}

//transform []byte to uint32
func toUint32(bytes []byte) uint32 {
	return (uint32(bytes[0]) << 24) + (uint32(bytes[1]) << 16) +
		(uint32(bytes[2]) << 8) + uint32(bytes[3])
}

func SHA1PRNG(seed []byte, size int) ([]byte, error) {
	hashs := SHA1(SHA1(seed))
	maxLen := len(hashs)
	realLen := size
	if realLen > maxLen {
		return nil, fmt.Errorf("Not Support %d, Only Support Lower then %d [% x]", realLen, maxLen, hashs)
	}
	return hashs[0:realLen], nil
}

func SHA1(data []byte) []byte {
	h := sha1.New()
	h.Write(data)
	return h.Sum(nil)
}

func SHA512(data []byte) []byte {
	h := sha512.New()
	h.Write(data)
	return h.Sum(nil)
}

//生成随机密钥，输出结果是经过base32编码
func GenerateSecretKey(seed string) (string, error) {
	coder := base64.NewEncoding(base64Table)
	decodeSeed, err := coder.DecodeString(seed)
	if err != nil {
		return "", err
	}
	// fmt.Println("decodeSeed:", hex.EncodeToString(decodeSeed))
	key, err := SHA1PRNG(decodeSeed, secretSize)
	if err != nil {
		return "", err
	}
	// fmt.Println("key:", hex.EncodeToString(key))
	maxLen := base32.StdEncoding.EncodedLen(len(key))
	ret := make([]byte, maxLen)
	base32.StdEncoding.Encode(ret, key)
	return string(ret), nil
}

/*
 * @生成code验证码
 * @secretKey 		密钥
 * @curTimeSlice 	当前时间戳 - time.Now().Unix()
 * @valid 			是否校验操作 true.是 false.否
 */
func generateCode(secretKey string, curTimeSlice int64, valid bool) uint32 {
	//base32 解码
	secretKeyBytes, _ := base32.StdEncoding.DecodeString(secretKey)
	//转换curTimeSlice为byte数组
	counterByBytes := toBytes(curTimeSlice)
	if !valid {
		counterByBytes = toBytes(curTimeSlice / 30)
	}
	//sign the value using hmac-sha1
	hmacSha1 := hmac.New(sha1.New, secretKeyBytes)
	hmacSha1.Write(counterByBytes)
	hash := hmacSha1.Sum(nil)
	//the next to get a subset of the generated hash as the password
	//choose the index:using the last nibble(half byte)to choose the index to start from
	offset := hash[len(hash)-1] & 0x0F //Due to hmac-sha1,len(hash)-1 = 19
	//get a 32-bit chunk from the hash starting at offset
	hashParts := hash[offset : offset+4]
	//ignore the most significant bit as per RFC 4226
	hashParts[0] = hashParts[0] & 0x7F
	//transform hashParts to uint32
	number := toUint32(hashParts)
	// size to 6 digits, one million is the first number with 7 digits so the remainder of the division will always return < 7 digits
	pwd := number % 1000000
	return pwd
}

/*
 * @生成code验证码
 * @secretKey 		密钥
 * @curTimeSlice 	当前时间戳 - time.Now().Unix()
 */
func GetNewCode(secretKey string, curTimeSlice int64) uint32 {
	return generateCode(secretKey, curTimeSlice, false)
}

/*
 * @校验code有效性
 * @secretKey 	密钥
 * @code		验证码
 * @size 		偏移时间 s >= 1 && s <= 17
 */
func ValidCode(secretKey string, code uint32, size ...int) bool {
	//conver uinx msec time into a 30 second "window"
	curTimeSlice := time.Now().Unix() / 30
	//window is used to check codes generated in the near past
	wsize := windowSize //可偏移时间，default=3, max=17
	if size != nil && len(size) == 1 {
		wsize = size[0]
	}
	if wsize < 1 || wsize > 17 {
		return false
	}
	for i := -wsize; i <= wsize; i++ {
		hash := generateCode(secretKey, (curTimeSlice + int64(i)), true)
		if hash == code {
			return true
		}
	}
	return false
}

/*
 * @生成密钥种子
 */
func GenerateSeed() string {
	hash := SHA512(toBytes(util.GetUUIDInt64()))
	coder := base64.NewEncoding(base64Table)
	ret := coder.EncodeToString(hash)
	return ret
}
