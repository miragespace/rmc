package util

import "net"

// GetFreeTCPPort asks the kernel for a free open port that is ready to use.
// Source: https://github.com/slimsag/freeport/blob/master/freeport.go
func GetFreeTCPPort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", ":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

func GetFreeUDPPort() (int, error) {
	addr, err := net.ResolveUDPAddr("udp", ":0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenUDP("udp", addr)
	if err != nil {
		return 0, err
	}
	port := l.LocalAddr().(*net.UDPAddr).Port
	l.Close()
	return port, nil
}
