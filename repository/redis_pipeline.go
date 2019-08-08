package repository

import (
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"sync"
)

type RedisPipeline struct {
	FlushAfter int
	Pool       *redis.Pool
}

func (rp *RedisPipeline) Pipeline(exitImmediately <-chan struct{}, done chan struct{}, send <-chan RedisCommand, errs chan error) {
	// Start a number of goroutines equivalent to the number of Redis
	// connections available
	numDigesters := rp.Pool.MaxActive
	wg := sync.WaitGroup{}
	wg.Add(numDigesters)

	for i := 0; i < numDigesters; i++ {
		go rp.startRedisWorkers(exitImmediately, &wg, send, errs)
	}

	wg.Wait()
	close(done)
}

func (rp *RedisPipeline) startRedisWorkers(exitImmediately <-chan struct{}, wg *sync.WaitGroup, send <-chan RedisCommand, errs chan error) {
	defer wg.Done()

	// Get a Redis connection from the pool and defer the close
	conn := rp.Pool.Get()
	defer func(conn redis.Conn, errs chan error) {
		if err := conn.Close(); err != nil {
			errs <- err
		}
	}(conn, errs)

	connDone := make(chan struct{})
	sendDone := make(chan struct{})

	recv := make(chan chan RedisResult, rp.FlushAfter)

	go rp.sendRedisCommands(exitImmediately, send, sendDone, recv, conn, errs)
	go rp.flushRedisResults(exitImmediately, sendDone, recv, conn, errs)
	go rp.receiveRedisResults(exitImmediately, recv, conn, connDone, errs)

	// Wait for the receive to complete before exiting the goroutine
	<-connDone
}

// Send and flush Redis commands
func (rp *RedisPipeline) sendRedisCommands(exitImmediately <-chan struct{}, send <-chan RedisCommand, sendDone chan struct{}, recv chan chan RedisResult, conn redis.Conn, errs chan error) {
	defer close(sendDone)

	for cmd := range send {
		if err := conn.Send(cmd.Name, cmd.Args...); err != nil {
			errs <- errors.Wrapf(err, "cannot send to Redis")
			return
		}

		// Flush the connection if the receive channel is full
		if len(recv) == cap(recv) {
			if err := conn.Flush(); err != nil {
				errs <- errors.Wrapf(err, "cannot flush Redis connection")
				return
			}
		}

		select {
		case recv <- cmd.Result:
		case <-exitImmediately:
			errs <- errors.New("send cancelled")
			return
		}
	}
}

// Flush Redis connection after send channel closes
func (rp *RedisPipeline) flushRedisResults(exitImmediately <-chan struct{}, sendDone <-chan struct{}, recv chan chan RedisResult, conn redis.Conn, errs chan error) {
	defer close(recv)

	select {
	case <-sendDone:
		if err := conn.Flush(); err != nil {
			errs <- errors.Wrapf(err, "cannot flush Redis connection")
			return
		}
	case <-exitImmediately:
		errs <- errors.New("flush cancelled")
		return
	}
}

// Receive the results of the Redis commands
func (rp *RedisPipeline) receiveRedisResults(exitImmediately <-chan struct{}, recv chan chan RedisResult, conn redis.Conn, connDone chan struct{}, errs chan error) {
	defer close(connDone)

	for ch := range recv {
		var result RedisResult
		result.Value, result.Err = conn.Receive()

		select {
		case ch <- result:
		case <-exitImmediately:
			errs <- errors.New("receive cancelled")
			return
		}

		if result.Err != nil {
			errs <- errors.Wrap(result.Err, "error received in Redis response")
			return
		}
	}
}
