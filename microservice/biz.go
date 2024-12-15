package main

import (
	"context"
)

type MyBizServer struct {
	UnimplementedBizServer
}

func NewMyBizServer() *MyBizServer {
	return &MyBizServer{}
}

func (bs *MyBizServer) Check(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (bs *MyBizServer) Add(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (bs *MyBizServer) Test(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}
