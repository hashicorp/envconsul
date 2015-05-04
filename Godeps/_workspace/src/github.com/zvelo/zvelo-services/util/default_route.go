package util

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
)

// DefaultRoute will read /proc/net/route and look for an IPv4 default
// gateway. IPv6 gateways will not be found. It will then return the gateway as
// an IP.
func DefaultRoute() (ip net.IP, err error) {
	if runtime.GOOS != "linux" {
		return ip, fmt.Errorf("not attempting to determine default gateway on non-linux OS")
	}

	file, err := os.Open("/proc/net/route")
	if err != nil {
		return
	}
	defer file.Close()
	return scanRouteFile(file)
}

func scanRouteFile(input io.Reader) (ip net.IP, err error) {
	scanner := bufio.NewScanner(input)
	defaultRoute := net.IPv4(0, 0, 0, 0)
	for scanner.Scan() {
		var iface string
		var destBytes, gatewayBytes []byte
		conversions, _ := fmt.Sscanf(scanner.Text(), "%s %x %x", &iface, &destBytes, &gatewayBytes)
		if conversions == 3 {
			dest := net.IP(destBytes)
			if dest.Equal(defaultRoute) {
				ip = net.IP(reverseBytes(gatewayBytes))
			}
		}
	}
	err = scanner.Err()
	return
}

func reverseBytes(slice []byte) []byte {
	result := make([]byte, len(slice), cap(slice))
	i := len(slice) - 1
	for _, b := range slice {
		result[i] = b
		i--
	}
	return result
}
