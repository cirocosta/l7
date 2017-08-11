package lib

import (
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandomUppercaseString(length int) string {
	const bigA int = 65
	const bigZ int = 90
	var res = make([]byte, length)

	for ndx, _ := range res {
		res[ndx] = byte(rand.Intn(bigZ-bigA) + bigA)
	}

	return string(res)
}

func RandomLowercaseString(length int) string {
	const lowerA int = 97
	const lowerZ int = 122
	var res = make([]byte, length)

	for ndx, _ := range res {
		res[ndx] = byte(rand.Intn(lowerZ-lowerA) + lowerA)
	}

	return string(res)
}
