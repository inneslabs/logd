package conn

import (
	"errors"
	"fmt"
	"net"
)

// GetConn returns a udp socket for logd
// using the value of LOGD_HOSTNAME
func GetConn(addr string) (net.Conn, error) {
	conn, err := net.Dial("udp", addr+":6102")
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	return conn, nil
}

func GetAddr(host string) (string, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", fmt.Errorf("lookup host err: %w", err)
	}
	if len(addrs) < 1 {
		return "", errors.New("no address found for host " + host)
	}
	return addrs[0], nil
}
