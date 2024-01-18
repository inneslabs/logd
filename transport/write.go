/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package transport

import (
	"errors"
	"fmt"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

func (t *Transporter) handleWrite(c *cmd.Cmd, unpk *auth.Unpacked) error {
	if c.Msg == nil {
		return errors.New("msg is nil")
	}
	valid, err := auth.Verify(t.writeSecret, unpk)
	if !valid || err != nil {
		return errors.New("unauthorised to write")
	}
	// marshal msg
	msgBytes, err := proto.Marshal(c.Msg)
	if err != nil {
		return fmt.Errorf("protobuf marshal msg err: %w", err)
	}
	// write to buffer
	t.buf.Write(msgBytes)
	// pipe to tails
	t.Out <- &ProtoPair{
		Msg:   c.Msg,
		Bytes: msgBytes,
	}
	// pipe to alarm svc
	t.alarmSvc.In <- c.Msg
	return nil
}
