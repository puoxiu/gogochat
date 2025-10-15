package cache

import (
	"time"
)

// Cache 定义缓存操作的通用接口
type Cache interface {
	SetKeyEx(key string, value string, timeout time.Duration) error
	GetKey(key string) (string, error)
	GetKeyNilIsErr(key string) (string, error)
	GetKeyWithPrefixNilIsErr(prefix string) (string, error)
	GetKeyWithSuffixNilIsErr(suffix string) (string, error)
	DelKeyIfExists(key string) error
	DelKeysWithPattern(pattern string) error
	DelKeysWithPrefix(prefix string) error
	DelKeysWithSuffix(suffix string) error
	DeleteAllRedisKeys() error
}

// 全局缓存实例
var myCache Cache

// Init 初始化全局缓存（需在程序启动时调用）
func Init(c Cache) {
	myCache = c
}

// GetGlobalCache 获取全局缓存实例
func GetGlobalCache() Cache {
	if myCache == nil {
		panic("cache not initialized: call Init() first")
	}
	return myCache
}