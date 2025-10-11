package random

import (
	"math"
	"math/rand"
	"strconv"
	"time"
)

// GetRandomInt 生成指定长度的随机整数（可用于验证码）
func GetRandomInt(len int) int {
	return rand.Intn(9*int(math.Pow(10, float64(len-1)))) + int(math.Pow(10, float64(len-1)))
}

func GetNowAndLenRandomString(len int) string {
	return time.Now().Format("20060102") + strconv.Itoa(GetRandomInt(len))
}
