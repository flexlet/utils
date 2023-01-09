package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func RandNum(max int) int {
	s := rand.NewSource(time.Now().UnixNano())
	r := rand.New(s)
	return r.Intn(max + 1)
}

func RandomString(len int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, len)
	rand.Read(b)
	return fmt.Sprintf("%x", b)[:len]
}
