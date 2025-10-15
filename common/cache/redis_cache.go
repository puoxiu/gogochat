package cache

import (
	"context"
	"strconv"
	"time"
	"fmt"
	// "log"
	"errors"
	"github.com/go-redis/redis/v8"
	"github.com/puoxiu/gogochat/pkg/zlog"
)


type RedisCache struct {
	client *redis.Client
	ctx    context.Context
}

func NewRedisCache(ctx context.Context, host string, port int, password string, db int) *RedisCache {
	addr := host + ":" + strconv.Itoa(port)
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	return &RedisCache{
		client: client,
		ctx:    ctx,
	}
}


func (rc *RedisCache)SetKeyEx(key string, value string, timeout time.Duration) error {
	err := rc.client.Set(rc.ctx, key, value, timeout).Err()
	if err != nil {
		return err
	}
	return nil
}

func (rc *RedisCache)GetKey(key string) (string, error) {
	value, err := rc.client.Get(rc.ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			zlog.Info("该key不存在")
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (rc *RedisCache)GetKeyNilIsErr(key string) (string, error) {
	value, err := rc.client.Get(rc.ctx, key).Result()
	if err != nil {
		return "", err
	}
	return value, nil
}

func (rc *RedisCache)GetKeyWithPrefixNilIsErr(prefix string) (string, error) {
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = rc.client.Keys(rc.ctx, prefix+"*").Result()
		if err != nil {
			return "", err
		}

		if len(keys) == 0 {
			zlog.Info("没有找到相关前缀key")
			return "", redis.Nil
		}

		if len(keys) == 1 {
			zlog.Info(fmt.Sprintln("成功找到了相关前缀key", keys))
			return keys[0], nil
		} else {
			zlog.Error("找到了数量大于1的key，查找异常")
			return "", errors.New("找到了数量大于1的key，查找异常")
		}
	}

}

func (rc *RedisCache)GetKeyWithSuffixNilIsErr(suffix string) (string, error) {
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = rc.client.Keys(rc.ctx, "*"+suffix).Result()
		if err != nil {
			return "", err
		}

		if len(keys) == 0 {
			zlog.Info("没有找到相关后缀key")
			return "", redis.Nil
		}

		if len(keys) == 1 {
			zlog.Info(fmt.Sprintln("成功找到了相关后缀key", keys))
			return keys[0], nil
		} else {
			zlog.Error("找到了数量大于1的key，查找异常")
			return "", errors.New("找到了数量大于1的key，查找异常")
		}
	}

}

func (rc *RedisCache)DelKeyIfExists(key string) error {
	exists, err := rc.client.Exists(rc.ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 1 { // 键存在
		delErr := rc.client.Del(rc.ctx, key).Err()
		if delErr != nil {
			return delErr
		}
	}
	// 无论键是否存在，都不返回错误
	return nil
}

func (rc *RedisCache)DelKeysWithPattern(pattern string) error {
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = rc.client.Keys(rc.ctx, pattern).Result()
		if err != nil {
			return err
		}

		// 如果没有更多的键，则跳出循环
		if len(keys) == 0 {
			// log.Println("没有找到对应key")
			break
		}

		// 删除找到的键
		if len(keys) > 0 {
			_, err = rc.client.Del(rc.ctx, keys...).Result()
			if err != nil {
				return err
			}
			// log.Println("成功删除相关对应key", keys)
		}
	}

	return nil
}

func (rc *RedisCache)DelKeysWithPrefix(prefix string) error {
	//var cursor uint64 = 0
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = rc.client.Keys(rc.ctx, prefix+"*").Result()
		if err != nil {
			return err
		}

		// 如果没有更多的键，则跳出循环
		if len(keys) == 0 {
			// log.Println("没有找到相关前缀key")
			break
		}

		// 删除找到的键
		if len(keys) > 0 {
			_, err = rc.client.Del(rc.ctx, keys...).Result()
			if err != nil {
				return err
			}
			// log.Println("成功删除相关前缀key", keys)
		}
	}

	return nil
}

func (rc *RedisCache)DelKeysWithSuffix(suffix string) error {
	//var cursor uint64 = 0
	var keys []string
	var err error

	for {
		// 使用 Keys 命令迭代匹配的键
		keys, err = rc.client.Keys(rc.ctx, "*"+suffix).Result()
		if err != nil {
			return err
		}

		// 如果没有更多的键，则跳出循环
		if len(keys) == 0 {
			// log.Println("没有找到相关后缀key")
			break
		}

		// 删除找到的键
		if len(keys) > 0 {
			_, err = rc.client.Del(rc.ctx, keys...).Result()
			if err != nil {
				return err
			}
			// log.Println("成功删除相关后缀key", keys)
		}
	}

	return nil
}

func (rc *RedisCache)DeleteAllRedisKeys() error {
	var cursor uint64 = 0
	for {
		keys, nextCursor, err := rc.client.Scan(rc.ctx, cursor, "*", 0).Result()
		if err != nil {
			return err
		}
		cursor = nextCursor

		if len(keys) > 0 {
			_, err := rc.client.Del(rc.ctx, keys...).Result()
			if err != nil {
				return err
			}
		}

		if cursor == 0 {
			break
		}
	}
	return nil
}


