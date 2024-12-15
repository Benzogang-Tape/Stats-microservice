package main

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sync"
	"time"
)

type MyAdminServer struct {
	UnimplementedAdminServer
	mu     sync.RWMutex
	events map[chan *Event]struct{}
	stat   []*Stat
}

func NewMyAdminServer() *MyAdminServer {
	return &MyAdminServer{
		mu:     sync.RWMutex{},
		events: make(map[chan *Event]struct{}),
		stat:   make([]*Stat, 0),
	}
}

func (as *MyAdminServer) Logging(n *Nothing, logSrv Admin_LoggingServer) error {
	newEventCh := make(chan *Event)
	as.mu.Lock()
	as.events[newEventCh] = struct{}{}
	as.mu.Unlock()

	for {
		select {
		case event, ok := <-newEventCh:
			if !ok {
				as.mu.Lock()
				delete(as.events, newEventCh)
				as.mu.Unlock()
				return status.Errorf(codes.Internal, "event stream closed")
			}

			if err := logSrv.Send(event); err != nil {
				return status.Errorf(codes.Internal, "sending event: %s", err.Error())
			}
			continue
		case <-logSrv.Context().Done():
		}
		break
	}

	as.mu.Lock()
	defer as.mu.Unlock()
	close(newEventCh)
	delete(as.events, newEventCh)

	return nil
}

func (as *MyAdminServer) Statistics(gap *StatInterval, statSrv Admin_StatisticsServer) error {
	ticker := time.NewTicker(time.Second * time.Duration(gap.GetIntervalSeconds()))
	defer ticker.Stop()
	statistics := &Stat{
		ByMethod:   make(map[string]uint64),
		ByConsumer: make(map[string]uint64),
	}

	as.mu.Lock()
	as.stat = append(as.stat, statistics)
	as.mu.Unlock()

	for {
		select {
		case <-ticker.C:
			err := as.sendStatistics(statistics, statSrv)
			if err != nil {
				return status.Errorf(codes.Internal, "sending statistics: %s", err.Error())
			}
			continue
		case <-statSrv.Context().Done():
		}
		break
	}

	return nil
}

func (as *MyAdminServer) sendStatistics(stat *Stat, outStream Admin_StatisticsServer) error {
	as.mu.Lock()
	defer as.mu.Unlock()

	stat.Timestamp = time.Now().Unix()
	if err := outStream.Send(stat); err != nil {
		return err
	}

	stat.ByConsumer = make(map[string]uint64)
	stat.ByMethod = make(map[string]uint64)

	return nil
}
