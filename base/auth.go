package base

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
)

func Authenticate(seed []byte, name, key string) []byte {
	res := make([]byte, 16)
	if len(seed) != 16 {
		rand.Read(res)
	} else {
		copy(res, seed)
	}
	h := hmac.New(sha256.New, []byte(key))
	h.Write(res)
	h.Write([]byte(name))
	res = append(res, h.Sum(nil)...)
	return res[:32]
}
