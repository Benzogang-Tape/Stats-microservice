package main

import (
	"fmt"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"net"
)

type Server struct {
	srv *grpc.Server
	lis net.Listener
}

func NewServer(addr string, acl ACL) (*Server, error) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on addrress %s: %w", addr, err)
	}

	adm := NewMyAdminServer()
	server := grpc.NewServer(
		grpc.InTapHandle(newInTapACLHandler(acl)),
		grpc.UnaryInterceptor(adm.newLoggingUnaryInterceptor()),
		grpc.StreamInterceptor(adm.newLoggingStreamInterceptor()),
	)
	RegisterBizServer(server, NewMyBizServer())
	RegisterAdminServer(server, adm)

	return &Server{
		srv: server,
		lis: lis,
	}, nil
}

func (s *Server) Start() error {
	eg := errgroup.Group{}

	eg.Go(func() error {
		return s.srv.Serve(s.lis)
	})

	return eg.Wait()
}

func (s *Server) Stop() {
	s.srv.GracefulStop()
}
