package crypto

import (
	"encoding/hex"

	"github.com/tjfoc/gmsm/sm3"
)

func SM3(data string) string {
	h := sm3.New()
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func EncryptPassword(data string) string {
	return SM3(data)
}

func VerifyPassword(cipherPwd, plainPwd string) bool {
	return cipherPwd == SM3(plainPwd)
}
