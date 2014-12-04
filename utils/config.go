package utils

import (
	"fmt"
	"net"
	"strings"
)

type Config struct {
	P string
	B []string
}

func (c Config) Ports() []int {
	var ports []int
	var begin, end int
	_, err := fmt.Sscanf(c.P, "%d-%d", &begin, &end)
	if err == nil {
		if begin < end {
			for p := begin; p <= end; p++ {
				ports = append(ports, p)
			}
		}
	}
	return ports
}

func (c Config) Bootstrap() []net.UDPAddr {
	var udpaddrs []net.UDPAddr
	for _, s := range c.B {
		z := strings.SplitN(s, ":", 2)
		if len(z) != 2 {
			continue
		}
		host := z[0]
		var begin, end uint16
		_, err := fmt.Sscanf(z[1], "%d-%d", &begin, &end)
		if err == nil && begin < end {
			addrs, err := net.LookupIP(host)
			if err == nil {
				for _, a := range addrs {
					for p := begin; p <= end; p++ {
						addr := net.UDPAddr{
							Port: int(p),
							IP:   a,
						}
						udpaddrs = append(udpaddrs, addr)
					}
				}
			}
		}
	}
	return udpaddrs
}

var DefaultConfig = Config{
	P: "9200-9210",
	B: []string{
		"localhost:9200-9210",
		"h2so5.net:9200-9210",
	},
}
