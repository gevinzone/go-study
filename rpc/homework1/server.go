package homework1

import (
	"context"
	"errors"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/message"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/serialization"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/serialization/json"
	"net"
	"reflect"
	"strconv"
	"time"
)

var errServiceNotFound = errors.New("找不到服务")

type Server struct {
	services    map[string]reflectionStub
	serializers []serialization.Serializer
}

func NewServer() *Server {
	res := &Server{
		services:    map[string]reflectionStub{},
		serializers: make([]serialization.Serializer, 32),
	}
	res.RegisterSerializer(json.Serializer{})
	return res
}

func (s *Server) MustRegister(service Service) {
	err := s.Register(service)
	if err != nil {
		panic(err)
	}
}

func (s *Server) Register(service Service) error {
	s.services[service.Name()] = reflectionStub{
		value:       reflect.ValueOf(service),
		serializers: s.serializers,
	}
	return nil
}

func (s *Server) RegisterSerializer(serializer serialization.Serializer) {
	s.serializers[serializer.Code()] = serializer
}

func (s *Server) Start(addr string) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go func() {
			if err := s.handleConn(conn); err != nil {
				_ = conn.Close()
				return
			}
		}()
	}
}

func (s *Server) handleConn(conn net.Conn) error {
	for {
		reqMsg, err := ReadMsg(conn)
		if err != nil {
			return err
		}
		req := message.DecodeReq(reqMsg)
		resp := &message.Response{
			Version:    req.Version,
			Compressor: req.Compressor,
			Serializer: req.Serializer,
			MessageId:  req.MessageId,
		}
		service, ok := s.services[req.ServiceName]
		writeErrorResp := func(err error) error {
			resp.Error = []byte(err.Error())
			resp.SetHeadLength()
			_, err = conn.Write(message.EncodeResp(resp))
			return err
		}
		if !ok {
			if err = writeErrorResp(errServiceNotFound); err != nil {
				return err
			}
			continue
		}
		ctx := context.Background()
		var cancel = func() {}
		for key, value := range req.Meta {
			if key == "timeout" {
				deadline, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					cancel()
					if err = writeErrorResp(err); err != nil {
						return err
					}
					continue
				}
				ctx, cancel = context.WithDeadline(ctx, time.UnixMilli(deadline))

			} else {
				ctx = context.WithValue(ctx, key, value)
			}
		}
		data, err := service.invoke(ctx, req)
		cancel()
		if req.Meta["oneway"] != "" {
			continue
		}
		if err != nil {
			// 返回客户端一个错误信息
			if err = writeErrorResp(err); err != nil {
				return err
			}
			continue
		}
		resp.SetHeadLength()
		resp.BodyLength = uint32(len(data))
		resp.Data = data
		data = message.EncodeResp(resp)
		_, err = conn.Write(data)
		if err != nil {
			return err
		}
	}
}

type reflectionStub struct {
	value       reflect.Value
	serializers []serialization.Serializer
}

func (r *reflectionStub) invoke(ctx context.Context, req *message.Request) ([]byte, error) {
	methodName := req.MethodName
	data := req.Data
	serializer := r.serializers[req.Serializer]
	if serializer == nil {
		// 返回客户端一个错误信息
		return nil, errors.New("micro: 不支持的序列化协议")
	}
	method := r.value.MethodByName(methodName)
	in := reflect.New(method.Type().In(1).Elem())
	err := serializer.Decode(data, in.Interface())
	if err != nil {
		return nil, err
	}
	res := method.Call([]reflect.Value{reflect.ValueOf(ctx), in})
	if len(res) > 1 && !res[1].IsZero() {
		return nil, res[1].Interface().(error)
	}
	return serializer.Encode(res[0].Interface())
}
