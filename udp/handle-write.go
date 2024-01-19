/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package udp

import (
	"errors"
	"fmt"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/cmd"
	"google.golang.org/protobuf/proto"
)

func (svc *UdpSvc) handleWrite(c *cmd.Cmd, unpk *auth.Unpacked) error {
	// verify authenticity
	valid, err := auth.Verify(svc.writeSecret, unpk)
	if !valid || err != nil {
		return errors.New("unauthorised to write")
	}
	// marshal msg
	msgBytes, err := proto.Marshal(c.Msg)
	if err != nil {
		return fmt.Errorf("protobuf marshal msg err: %w", err)
	}
	// write to buffer
	svc.buf.Write(msgBytes)
	// pipe to tails
	svc.forSubs <- &ProtoPair{
		Msg:   c.Msg,
		Bytes: msgBytes,
	}
	// pipe to alarm svc
	svc.alarmSvc.In <- c.Msg
	return nil
}
