package homework1

import (
	"context"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/message"
)

type Proxy interface {
	Invoke(ctx context.Context, req *message.Request) (*message.Response, error)
}
