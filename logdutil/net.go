package logdutil

import (
	"errors"
	"fmt"
	"net"
	"os"
)

var logdHost string

func init() {
	logdHost = os.Getenv("LOGD_HOST")
}

func GetConn(host *string) net.Conn {
	var (
		addr string
		err  error
	)
	if host == nil {
		addr, err = getAddr(logdHost)
	} else {
		addr, err = getAddr(*host)
	}
	if err != nil {
		panic("get addr err: " + err.Error())
	}
	conn, err := net.Dial("udp", addr+":6102")
	if err != nil {
		panic("failed to connect: " + err.Error())
	}
	return conn
}

func getAddr(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", fmt.Errorf("lookup host err: %w", err)
	}
	if len(addrs) < 1 {
		return "", errors.New("no address found for host " + host)
	}
	return addrs[0], nil
}
