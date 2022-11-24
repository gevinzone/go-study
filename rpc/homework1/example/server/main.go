package main

import (
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/serialization/json"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/serialization/proto"
)

func main() {
	server := homework1.NewServer()
	server.MustRegister(&UserService{})
	server.MustRegister(&UserServiceProto{})
	server.RegisterSerializer(json.Serializer{})
	server.RegisterSerializer(proto.Serializer{})

	if err := server.Start(":8081"); err != nil {
		panic(err)
	}
}
