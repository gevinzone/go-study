package lock

import (
	"context"
	_ "embed"
	"errors"
	"github.com/go-redis/redis/v9"
	"github.com/google/uuid"
	"sync"
	"time"
)

var (
	//go:embed lua/lock.lua
	luaLock string
	//go:embed lua/unlock.lua
	luaUnlock string
	//go:embed lua/refresh.lua
	luaRefresh string

	ErrFailedToPreemptLock = errors.New("rlock: 抢锁失败")
	ErrLockNotHold         = errors.New("rlock: 未持有锁")
	ErrFailToRefreshLock   = errors.New("rlock: 锁刷新失败")
)

type Client struct {
	client redis.Cmdable
}

func NewClient(cmd redis.Cmdable) *Client {
	return &Client{client: cmd}
}

func (c *Client) Lock(ctx context.Context, key string,
	expiration time.Duration, retry RetryStrategy, timeout time.Duration) (*Lock, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	val := uuid.New().String()
	if l, err := c.createLock(ctx, key, val, expiration, timeout); l != nil || errors.Is(err, context.DeadlineExceeded) {
		return l, err
	}

	for {
		// 超时，或没抢到锁，则重试
		interval, ok := retry.Next()
		if !ok {
			return nil, ErrFailedToPreemptLock
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
			if l, _ := c.createLock(ctx, key, val, expiration, timeout); l != nil {
				return l, nil
			}
		}
	}

}

func (c *Client) createLock(ctx context.Context, key string, val string,
	expiration time.Duration, timeout time.Duration) (*Lock, error) {
	lctx, cancel := context.WithTimeout(ctx, timeout)
	res, err := c.client.Eval(lctx, luaLock, []string{key},
		val, expiration.Seconds()).Result()
	cancel()
	if res == "OK" {
		return &Lock{
			client:     c.client,
			value:      val,
			key:        key,
			expiration: expiration,
			unlock:     make(chan struct{}, 1),
		}, nil
	}
	return nil, err
}

type Lock struct {
	client     redis.Cmdable
	value      string
	key        string
	expiration time.Duration

	unlock     chan struct{}
	unlockOnce sync.Once
}

func (l *Lock) AutoRefresh(interval time.Duration, timeout time.Duration, retry RetryStrategy) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	retrySignal := make(chan struct{}, 1)
	defer close(retrySignal)
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			err := l.Refresh(ctx)
			cancel()
			if err == context.DeadlineExceeded {
				retrySignal <- struct{}{}
				continue
			}
			if err != nil {
				return err
			}
		case <-retrySignal:
			rInterval, ok := retry.Next()
			if !ok {
				return ErrFailToRefreshLock
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			err := l.Refresh(ctx)
			cancel()
			if err == context.DeadlineExceeded {
				time.Sleep(rInterval)
				retrySignal <- struct{}{}
				continue
			}
			if err != nil {
				return err
			}
		case <-l.unlock:
			return nil
		}
	}
}

func (l *Lock) Refresh(ctx context.Context) error {
	res, err := l.client.Eval(ctx, luaRefresh,
		[]string{l.key}, l.value, l.expiration.Seconds()).Int64()
	if err != nil {
		return err
	}
	if res != 1 {
		return ErrLockNotHold
	}
	return nil
}

func (l *Lock) Unlock(ctx context.Context) error {
	l.unlockOnce.Do(func() {
		close(l.unlock)
	})
	res, err := l.client.Eval(ctx, luaUnlock, []string{l.key},
		l.value).Int64()
	if err != nil {
		return err
	}
	if res != 1 {
		return ErrLockNotHold
	}
	return nil
}
