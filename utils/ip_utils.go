package utils

import "net"

func GetReplyIP(clientAddr *net.UDPAddr) net.IP {
	conn, err := net.DialUDP("udp", nil, clientAddr)
	if err != nil {
		return net.ParseIP("127.0.0.1")
	}
	defer conn.Close()

	return conn.LocalAddr().(*net.UDPAddr).IP
}
