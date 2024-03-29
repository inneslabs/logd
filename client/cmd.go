package client

import (
	"context"
	"fmt"

	"github.com/inneslabs/logd/auth"
	"github.com/inneslabs/logd/cmd"
	"google.golang.org/protobuf/proto"
)

func (client *Client) Cmd(ctx context.Context, command *cmd.Cmd, secret []byte) error {
	payload, err := proto.Marshal(command)
	if err != nil {
		return fmt.Errorf("err marshalling cmd: %w", err)
	}
	signed, err := auth.Sign(secret, payload)
	if err != nil {
		return fmt.Errorf("err signing cmd: %w", err)
	}
	if client.rateLimiter != nil {
		client.rateLimiter.Wait(ctx)
	}
	client.conn.Write(signed)
	return nil
}
