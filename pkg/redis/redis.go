package redis

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"time"
)

var (
	Redis *redis.Client
)

const DEFAULT_EXPIRE = time.Hour * 2

func InitRedis(addr, password string, db int) error {
	Redis = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	_, err := Redis.Ping().Result()
	if err != nil {
		fmt.Println("初始化Redis失败,msg=", err.Error())
		return err
	}
	return nil
}

func Set(key string, value interface{}, expireTime time.Duration) error {
	if Redis == nil {
		return errors.New("redis is not initialized")
	}
	v, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Redis.Set(key, string(v), expireTime).Err()
}

func SetDefault(key string, value interface{}) error {
	if Redis == nil {
		return errors.New("redis is not initialized")
	}
	v, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return Redis.Set(key, string(v), DEFAULT_EXPIRE).Err()
}

func Get(key string, v interface{}) (bool, error) {
	if Redis == nil {
		return false, errors.New("redis is not initialized")
	}
	isExists, err := exists(key)
	if err != nil {
		return false, err
	}
	if !isExists {
		return false, nil
	}
	value, err := Redis.Get(key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, err
	}
	if err := json.Unmarshal([]byte(value), v); err != nil {
		return false, err
	}
	return true, nil
}

func Del(key string) error {
	if Redis == nil {
		return errors.New("redis is not initialized")
	}
	_, err := Redis.Del(key).Result()
	if err != nil {
		return err
	}
	return nil
}

func Exists(key string) (bool, error) {
	if Redis == nil {
		return false, errors.New("redis is not initialized")
	}
	return exists(key)
}

func Keys(key string) (err error, keys []string) {
	if Redis == nil {
		return errors.New("redis is not initialized"), nil
	}
	keys, err = Redis.Keys(key).Result()
	if err != nil {
		return err, nil
	}
	return nil, keys
}

func Dels(key string) error {
	err, keys := Keys(key)
	if err != nil {
		return err
	}
	for i := 0; i < len(keys); i++ {
		err := Del(keys[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func exists(key string) (bool, error) {
	result, err := Redis.Exists(key).Result()
	if err != nil {
		return false, err
	}
	if result == 0 {
		return false, nil
	}
	return true, nil
}
