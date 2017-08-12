package lib

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	bigA   int = 65
	bigZ   int = 90
	lowerA int = 97
	lowerZ int = 122
)

func RandomUppercaseString(length int) string {
	var res = make([]byte, length)

	for ndx, _ := range res {
		res[ndx] = byte(rand.Intn(bigZ-bigA) + bigA)
	}

	return string(res)
}

func RandomLowercaseString(length int) string {
	var res = make([]byte, length)

	for ndx, _ := range res {
		res[ndx] = byte(rand.Intn(lowerZ-lowerA) + lowerA)
	}

	return string(res)
}
