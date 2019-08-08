package repository

import (
	"github.com/alicebob/miniredis"
	"github.com/fortytw2/leaktest"
	"github.com/gomodule/redigo/redis"
	"testing"
	"time"
)

func TestNewRedisPool(t *testing.T) {
	defer leaktest.Check(t)()

	t.Run("should return a new Redis pool with sensible defaults", func(t *testing.T) {
		pool := NewRedisPool()
		conn := pool.Get()
		resp, err := redis.String(conn.Do("PING"))
		if err != nil {
			if err.Error() != "dial tcp :6379: connect: connection refused" {
				t.Errorf("connections from Redis pool should connect to `%s` by default", ":6379")
			}
			return
		}

		if resp != "PONG" {
			t.Errorf("got `%s`, want `%s` from local Redis server", resp, "PONG")
		}
	})

	t.Run("should set options as provided", func(t *testing.T) {
		s, err := miniredis.Run()
		if err != nil {
			t.Fatal(err)
		}
		defer s.Close()

		options := []RedisPoolOption{
			RedisPoolDial(func() (redis.Conn, error) {
				return redis.Dial("tcp", s.Addr())
			}),
			RedisPoolIdleTimeout(42 * time.Second),
			RedisPoolMaxActive(42),
			RedisPoolMaxConnLifetime(42 * time.Minute),
			RedisPoolMaxIdle(24),
			RedisPoolTestOnBorrow(func(c redis.Conn, tm time.Time) error {
				_, err := c.Do("PING")
				return err
			}),
			RedisPoolWait(false),
		}

		pool := NewRedisPool(options...)

		if pool.IdleTimeout != 42*time.Second {
			t.Errorf("got `%v`, want `%v` for pool IdleTimeout", pool.IdleTimeout, 42*time.Second)
		}

		if pool.MaxActive != 42 {
			t.Errorf("got `%d`, want `%d` for pool MaxActive", pool.MaxActive, 42)
		}

		if pool.MaxConnLifetime != 42*time.Minute {
			t.Errorf("got `%v`, want `%v` for pool MaxConnLifetime", pool.MaxConnLifetime, 42*time.Minute)
		}

		if pool.MaxIdle != 24 {
			t.Errorf("got `%d`, want `%d` for pool MaxIdle", pool.MaxIdle, 24)
		}

		if pool.Wait != false {
			t.Errorf("got `%v`, want `%v` for pool Wait", pool.Wait, false)
		}

		conn := pool.Get()
		if _, err := redis.String(conn.Do("GET", "foo")); err != redis.ErrNil {
			t.Error(err)
		}

		if err := pool.TestOnBorrow(conn, time.Now()); err != nil {
			t.Error(err)
		}
	})
}
