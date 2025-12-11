package register

import (
	"net"
	"strings"
)

// AutoDetectIP finds an available private IPv4 address
func autoDetectIP() string {
	as, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}

	for _, a := range as {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP.String()

		// Skip loopback
		if ipnet.IP.IsLoopback() {
			continue
		}

		// Only IPv4
		if ipnet.IP.To4() == nil {
			continue
		}

		// Filter typical private ranges (docker/cni included)
		if strings.HasPrefix(ip, "172.") || strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") {
			return ip
		}
	}

	return ""
}
