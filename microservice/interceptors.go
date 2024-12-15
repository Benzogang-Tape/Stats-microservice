package main

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/tap"
	"regexp"
	"time"
)

type ACL map[string][]string

const (
	consumerKey = "consumer"

	ErrACLParsingFailed = "failed to parse ACL JSON"
	MsgNoMetaData       = "no metadata in context"
	MsgNoConsumerInfo   = "no consumer info"
	MsgNoConsumerInACL  = "consumer is not in the acl"
	MsgMethodNotAllowed = "method not allowed"
	MsgUnknownHost      = "unknown host"
)

func newInTapACLHandler(acl ACL) func(ctx context.Context, info *tap.Info) (context.Context, error) {
	return func(ctx context.Context, info *tap.Info) (context.Context, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, MsgNoMetaData)
		}

		consumers, ok := md[consumerKey]
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, MsgNoConsumerInfo)
		}

		consumer := consumers[0]
		allowedMethods, ok := acl[consumer]
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, MsgNoConsumerInACL)
		}

		for _, method := range allowedMethods {
			if matched, err := regexp.MatchString(method, info.FullMethodName); matched && (err == nil) {
				return ctx, nil
			}
		}

		return nil, status.Errorf(codes.Unauthenticated, MsgMethodNotAllowed)
	}
}

func (as *MyAdminServer) newLoggingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Errorf(codes.Internal, MsgNoMetaData)
		}

		p, ok := peer.FromContext(ss.Context())
		if !ok {
			return status.Errorf(codes.Unauthenticated, MsgUnknownHost)
		}

		consumer := md[consumerKey][0]
		as.updateStats(consumer, info.FullMethod)
		as.sendEvent(consumer, info.FullMethod, p.Addr.String())

		return handler(srv, ss)
	}
}

func (as *MyAdminServer) newLoggingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Internal, MsgNoMetaData)
		}

		p, ok := peer.FromContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Unauthenticated, MsgUnknownHost)
		}

		consumer := md[consumerKey][0]
		as.updateStats(consumer, info.FullMethod)
		as.sendEvent(consumer, info.FullMethod, p.Addr.String())

		return handler(ctx, req)
	}
}

func (as *MyAdminServer) sendEvent(consumer, method, host string) {
	as.mu.Lock()
	defer as.mu.Unlock()

	for eventCh := range as.events {
		eventCh <- &Event{
			Timestamp: time.Now().Unix(),
			Consumer:  consumer,
			Method:    method,
			Host:      host,
		}
	}
}

func (as *MyAdminServer) updateStats(consumer, method string) {
	as.mu.Lock()
	defer as.mu.Unlock()

	for _, stat := range as.stat {
		stat.ByConsumer[consumer]++
		stat.ByMethod[method]++
	}
}
