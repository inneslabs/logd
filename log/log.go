package log

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/swissinfo-ch/logd/auth"
	"github.com/swissinfo-ch/logd/conn"
	"github.com/swissinfo-ch/logd/msg"
)

var (
	logdConn net.Conn
	env      = os.Getenv("SWI_ENV")
	svc      = os.Getenv("SWI_SVC")
	fn       = os.Getenv("SWI_FN")
	secret   []byte
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
	if logdConn == nil {
		logdConn = conn.GetConn()
	}

	// build msg
	msg := &msg.Msg{
		Timestamp: time.Now().UnixNano(),
		Env:       env,
		Svc:       svc,
		Fn:        fn,
		Lvl:       string(lvl),
		Msg:       fmt.Sprintf(template, args...),
	}

	// get ephemeral signature
	signedMsg, err := auth.Sign(secret, msg, time.Now())
	if err != nil {
		fmt.Println("logd.log sign msg err:", err)
		return
	}

	// write to socket
	_, err = logdConn.Write(signedMsg)
	if err != nil {
		fmt.Println("logd.log write udp err:", err)
	}
}
