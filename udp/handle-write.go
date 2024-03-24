package udp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/inneslabs/logd/auth"
	"github.com/inneslabs/logd/cmd"
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
	// write to store
	msgKey := c.Msg.GetKey()
	segments := strings.Split(msgKey, "/") // /prod/swi-core/SMRotation
	if len(segments) < 3 {
		return errors.New("invalid key")
	}
	storeKey := fmt.Sprintf("/%s/%s", segments[1], segments[2])
	svc.logStore.Write(storeKey, msgBytes)
	// send to tails
	svc.forSubs <- &ProtoPair{
		Msg:   c.Msg,
		Bytes: msgBytes,
	}
	return nil
}
