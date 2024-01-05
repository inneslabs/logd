package log

import (
	"errors"
	"fmt"
	"net"
	"os"
)

var logdHostname = os.Getenv("LOGD_HOSTNAME")

// GetConn returns a udp socket for logd
// using the value of LOGD_HOSTNAME
func GetConn() net.Conn {
	addr, err := getAddr(logdHostname)
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
