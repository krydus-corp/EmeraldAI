/*
 * File: redis.go
 * Project: redis
 * File Created: Monday, 18th October 2021 1:41:36 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package cache

import (
	"context"
	"fmt"
	"time"

	redis "github.com/go-redis/redis/v8"
)

type Cache struct {
	client *redis.Client
}

type Config struct {
	Host string
	Port int
}

func Initialize(cfg *Config) (*Cache, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})

	_, err := client.Ping(context.TODO()).Result()
	if err != nil {
		return nil, err
	}

	return &Cache{client: client}, nil
}

func (c *Cache) Set(key string, value interface{}, exp time.Duration) error {
	return c.client.Set(context.TODO(), key, value, exp).Err()
}

func (c *Cache) Get(key string) (string, error) {
	return c.client.Get(context.TODO(), key).Result()
}

func (c *Cache) Delete(key string) (int64, error) {
	return c.client.Del(context.TODO(), key).Result()
}

func (c *Cache) Expire(key string, exp time.Duration) (bool, error) {
	return c.client.Expire(context.TODO(), key, exp).Result()
}
func (c *Cache) Shutdown() {
	c.client.Shutdown(context.TODO())
}
