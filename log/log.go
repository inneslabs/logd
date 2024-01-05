package log

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/fxamacker/cbor/v2"
	"github.com/swissinfo-ch/logd/msg"
)

var (
	conn   net.Conn
	env    = os.Getenv("SWI_ENV")
	svc    = os.Getenv("SWI_SVC")
	fn     = os.Getenv("SWI_FN")
	secret []byte
)

type Lvl string

const (
	Error = Lvl("ERROR")
	Warn  = Lvl("WARN")
	Info  = Lvl("INFO")
	Debug = Lvl("DEBUG")
	Trace = Lvl("TRACE")
)

// SetSecret sets secret used to sign logs
// When calling Log
func SetSecret(s []byte) {
	secret = s
}

// Log writes a logd entry to machine at LOGD_HOSTNAME
// Values of SWI_ENV, SWI_SVC & SWI_FN are used
func Log(lvl Lvl, template string, args ...interface{}) {
	if conn == nil {
		conn = GetConn()
	}

	// marshal payload
	payload, err := cbor.Marshal(&msg.Msg{
		Timestamp: time.Now().UnixNano(),
		Env:       env,
		Svc:       svc,
		Fn:        fn,
		Lvl:       string(lvl),
		Msg:       fmt.Sprintf(template, args...),
	})
	if err != nil {
		fmt.Println("logd.log cbor marshal err:", err)
		return
	}

	// get ephemeral signature
	signed := Sign(secret, payload, time.Now())
	_, err = conn.Write(signed)
	if err != nil {
		fmt.Println("logd.log write udp err:", err)
	}
}
