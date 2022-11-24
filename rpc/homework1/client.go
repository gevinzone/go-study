package homework1

import (
	"context"
	"errors"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/message"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/serializer/json"
	"github.com/silenceper/pool"
	"net"
	"reflect"
	"strconv"
	"sync/atomic"
	"time"
)

var messageId uint32 = 0

type Client struct {
	connPool   pool.Pool
	serializer serialization.Serializer
}

func NewClient(addr string) (*Client, error) {
	p, err := pool.NewChannelPool(&pool.Config{
		InitialCap: 10,
		MaxCap:     100,
		MaxIdle:    50,
		Factory: func() (interface{}, error) {
			return net.Dial("tcp", addr)
		},
		IdleTimeout: time.Minute,
		Close: func(i interface{}) error {
			return i.(net.Conn).Close()
		},
	})
	if err != nil {
		return nil, err
	}
	return &Client{
		connPool:   p,
		serializer: json.Serializer{},
	}, nil
}

func (c *Client) Invoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	var (
		resp *message.Response
		err  error
	)
	ch := make(chan struct{})
	go func() {
		resp, err = c.doInvoke(ctx, req)
		close(ch)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch:
		return resp, err
	}
}

func (c *Client) doInvoke(ctx context.Context, req *message.Request) (*message.Response, error) {
	obj, err := c.connPool.Get()
	if err != nil {
		return nil, err
	}
	conn := obj.(net.Conn)
	data := message.EncodeReq(req)
	i, err := conn.Write(data)
	if err != nil {
		return nil, err
	}
	if i != len(data) {
		return nil, errors.New("micro: 未写入全部数据")
	}
	respMsg, err := ReadMsg(conn)
	if err != nil {
		return nil, err
	}
	return message.DecodeResp(respMsg), nil
}

func (c *Client) InitService(service Service) error {
	typ := reflect.TypeOf(service).Elem()
	val := reflect.ValueOf(service).Elem()
	numField := typ.NumField()
	for i := 0; i < numField; i++ {
		fieldType := typ.Field(i)
		fieldVal := val.Field(i)
		if !fieldVal.CanSet() {
			continue
		}
		fn := reflect.MakeFunc(fieldType.Type, func(args []reflect.Value) (results []reflect.Value) {
			outType := fieldType.Type.Out(0)
			ctx := args[0].Interface().(context.Context)
			arg := args[1].Interface()

			bs, err := c.serializer.Encode(arg)
			if err != nil {
				results = append(results, reflect.Zero(outType))
				results = append(results, reflect.ValueOf(err))
				return
			}
			msgId := atomic.AddUint32(&messageId, 1)
			meta := make(map[string]string, 2)
			if isOneway(ctx) {
				meta["oneway"] = "true"
			}
			deadline, ok := ctx.Deadline()
			if ok {
				meta["timeout"] = strconv.FormatInt(deadline.UnixMilli(), 10)
			}
			req := &message.Request{
				BodyLength:  uint32(len(bs)),
				MessageId:   msgId,
				Version:     0,
				Compressor:  0,
				Serializer:  c.serializer.Code(),
				ServiceName: service.Name(),
				MethodName:  fieldType.Name,
				Meta:        meta,
				Data:        bs,
			}
			req.CalHeadLength()
			resp, err := c.Invoke(ctx, req)
			//if isOneway(ctx) {
			//	return
			//}
			if err != nil {
				results = append(results, reflect.Zero(outType), reflect.ValueOf(err))
				return
			}
			resObj := reflect.New(outType).Interface()
			err = c.serializer.Decode(resp.Data, resObj)
			var errVal reflect.Value
			results = append(results, reflect.ValueOf(resObj).Elem())
			if err != nil {
				errVal = reflect.ValueOf(err)
			} else {
				errVal = reflect.Zero(reflect.TypeOf(new(error)).Elem())
			}
			results = append(results, errVal)
			return
		})
		fieldVal.Set(fn)
	}
	return nil
}

type Service interface {
	Name() string
}
