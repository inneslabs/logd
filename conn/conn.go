/*
Copyright © 2024 JOSEPH INNES <avianpneuma@gmail.com>
*/
package conn

import (
	"errors"
	"fmt"
	"net"
)

type Addr string

// Dial returns a udp socket for logd
func Dial(addr Addr) (net.Conn, error) {
	conn, err := net.Dial("udp", string(addr)+":6102")
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	return conn, nil
}

// Get aadr
func GetAddr(host string) (Addr, error) {
	addrs, err := net.LookupHost(host)
	if err != nil {
		return "", fmt.Errorf("lookup host err: %w", err)
	}
	if len(addrs) < 1 {
		return "", errors.New("no address found for host " + host)
	}
	return Addr(addrs[0]), nil
}
