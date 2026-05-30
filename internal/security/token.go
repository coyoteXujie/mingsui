package security

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const DefaultTokenBytes = 32

func GenerateToken(byteLen int) (string, error) {
	if byteLen < 16 {
		return "", fmt.Errorf("token 随机字节长度不能小于 16")
	}
	if byteLen > 128 {
		return "", fmt.Errorf("token 随机字节长度不能大于 128")
	}

	buf := make([]byte, byteLen)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
