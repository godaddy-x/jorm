package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
)

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(data []byte) []byte {
	defer func() {
		if r := recover(); r != nil {
			// log.Debug("Aes解密失败", 0, log.String("src", string(src)))
		}
	}()
	length := len(data)
	unpadding := int(data[length-1])
	return data[:(length - unpadding)]
}

func AesEncrypt(c, k string) string {
	if len(k) == 0 {
		return ""
	}
	if len(k) != 16 {
		return ""
	}
	key := []byte(k)
	data := []byte(c)
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	blockSize := block.BlockSize()
	data = PKCS5Padding(data, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	crypted := make([]byte, len(data))
	blockMode.CryptBlocks(crypted, data)
	return Base64Encode(string(crypted))
}

func AesDecrypt(c, k string) string {
	if len(k) == 0 {
		return ""
	}
	if len(k) != 16 {
		return ""
	}
	key := []byte(k)
	crypted := Base64Decode(c)
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	data := make([]byte, len(crypted))
	blockMode.CryptBlocks(data, []byte(crypted))
	data = PKCS5UnPadding(data)
	if data == nil {
		return ""
	}
	return string(data)
}
