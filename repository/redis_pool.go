package repository

import (
	"github.com/gomodule/redigo/redis"
	"time"
)

type RedisPoolOption struct {
	f func(*redis.Pool)
}

func RedisPoolDial(f func() (redis.Conn, error)) RedisPoolOption {
	return RedisPoolOption{func(do *redis.Pool) {
		do.Dial = f
	}}
}

func RedisPoolIdleTimeout(timeout time.Duration) RedisPoolOption {
	return RedisPoolOption{func(do *redis.Pool) {
		do.IdleTimeout = timeout
	}}
}

func RedisPoolMaxActive(i int) RedisPoolOption {
	return RedisPoolOption{func(do *redis.Pool) {
		do.MaxActive = i
	}}
}

func RedisPoolMaxConnLifetime(lifetime time.Duration) RedisPoolOption {
	return RedisPoolOption{func(do *redis.Pool) {
		do.MaxConnLifetime = lifetime
	}}
}

func RedisPoolMaxIdle(i int) RedisPoolOption {
	return RedisPoolOption{func(do *redis.Pool) {
		do.MaxIdle = i
	}}
}

func RedisPoolTestOnBorrow(f func(c redis.Conn, t time.Time) error) RedisPoolOption {
	return RedisPoolOption{func(do *redis.Pool) {
		do.TestOnBorrow = f
	}}
}

func RedisPoolWait(b bool) RedisPoolOption {
	return RedisPoolOption{func(do *redis.Pool) {
		do.Wait = b
	}}
}

func NewRedisPool(options ...RedisPoolOption) *redis.Pool {
	pool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", ":6379")
		},
	}

	for _, option := range options {
		option.f(pool)
	}

	return pool
}
