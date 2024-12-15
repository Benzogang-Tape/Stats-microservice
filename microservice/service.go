package main

import (
	context "context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"log/slog"
)

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
// если хочется, то для красоты можно разнести логику по разным файликам

func StartMyMicroservice(ctx context.Context, addr string, acl string) error {
	parsedACL, err := parseACL(acl)
	if err != nil {
		return errors.Wrap(err, ErrACLParsingFailed)
	}

	server, err := NewServer(addr, parsedACL)
	if err != nil {
		return errors.Wrap(err, "Failed to init server")
	}

	go func() {
		if err := server.Start(); err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Failed to start server: %s", err.Error()))
		}
	}()

	go func() {
		<-ctx.Done()
		server.Stop()
	}()

	return nil
}

func parseACL(acl string) (ACL, error) {
	parsedACL := make(ACL)
	if err := json.Unmarshal([]byte(acl), &parsedACL); err != nil {
		return nil, err
	}

	return parsedACL, nil
}
