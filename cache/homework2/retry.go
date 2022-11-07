package lock

import "time"

type RetryStrategy interface {
	// Next 返回下一次重试的时间间隔，如果不继续重试，第二个参数返回false
	Next() (time.Duration, bool)
}

type FixIntervalRetry struct {
	Interval time.Duration
	Max      int
	cnt      int
}

func (f *FixIntervalRetry) next() (time.Duration, bool) {
	f.cnt++
	return f.Interval, f.cnt <= f.Max
}
