package main

import (
	"context"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/example/proto/gen"
)

type UserServiceProto struct {
}

func (u *UserServiceProto) GetById(ctx context.Context, req *gen.GetByIdReq) (*gen.GetByIdResp, error) {
	return &gen.GetByIdResp{
		User: &gen.User{
			Id: 123,
		},
	}, nil
}

func (u *UserServiceProto) Name() string {
	return "user-service-proto"
}
