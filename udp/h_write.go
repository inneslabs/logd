package udp

import (
	"fmt"
	"strings"

	"github.com/inneslabs/logd/cmd"
	"github.com/inneslabs/logd/pkg"
	"google.golang.org/protobuf/proto"
)

func (svc *UdpSvc) handleWrite(c *cmd.Cmd, p *pkg.Pkg) {
	if !svc.guard.Good([]byte(svc.secrets.Write), p) {
		return
	}
	msgBytes, err := proto.Marshal(c.Msg)
	if err != nil {
		return
	}
	msgKey := c.Msg.GetKey()
	segments := strings.Split(msgKey, "/")
	if len(segments) < 3 {
		return
	}
	storeKey := fmt.Sprintf("/%s/%s", segments[1], segments[2])
	svc.logStore.Write(storeKey, msgBytes)
	svc.forSubs <- &ProtoPair{
		Msg:   c.Msg,
		Bytes: msgBytes,
	}
}