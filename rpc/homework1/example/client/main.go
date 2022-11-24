package main

import (
	"context"
	"encoding/json"
	"fmt"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1"
	"gitee.com/geektime-geekbang/geektime-go/rpc/homework1/example/proto/gen"
)

func main() {
	var (
		c   *homework1.Client
		err error
	)
	fmt.Println("starting...")
	if c, err = homework1.NewClient("0.0.0.0:8081"); err != nil {
		panic(err)
	}
	us := &UserService{}
	if err = c.InitService(us); err != nil {
		panic(err)
	}
	resp, err := us.GetById(context.Background(), &FindByUserIdReq{Id: 12})
	if err != nil {
		panic(err)
	}
	//_, _ = us.GetById(homework1.CtxWithOneway(context.Background()), &FindByUserIdReq{Id: 12})
	data, _ := json.Marshal(resp)
	fmt.Printf("收到响应: %s \n", data)

	_, err = us.AlwaysError(context.Background(), &FindByUserIdReq{Id: 12})
	fmt.Printf("收到错误信息: %s \n", err.Error())

	usProto := &UserServiceProto{}
	if err = c.InitService(usProto); err != nil {
		panic(err)
	}
	presp, err := usProto.GetById(context.Background(), &gen.GetByIdReq{Id: 12})
	if err != nil {
		panic(err)
	}
	data, _ = json.Marshal(presp)
	fmt.Printf("收到响应: %s \n", data)

}

type UserService struct {
	GetById     func(ctx context.Context, req *FindByUserIdReq) (*FindByUserIdResp, error)
	AlwaysError func(ctx context.Context, req *FindByUserIdReq) (*FindByUserIdResp, error)
}

func (u *UserService) Name() string {
	return "user"
}

type UserServiceProto struct {
	GetById func(ctx context.Context, req *gen.GetByIdReq) (*gen.GetByIdResp, error)
}

func (u *UserServiceProto) Name() string {
	return "user-service-proto"
}
