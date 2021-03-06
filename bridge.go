package hue

import (
	"bytes"
	"net"
	"strings"
	"time"
)

const (
	// Hue docs say to use "IpBridge" over "hue-bridgeid"
	_SSDPIdentifier = "IpBridge"

	_DefaultBufferSize = 256
	_DefaultTimeout    = 30 * time.Second
	_DefaultNumBridges = 8
)

var _SSDPData = []string{
	"M-SEARCH * HTTP/1.1",
	"HOST:239.255.255.250:1900",
	"MAN:\"ssdp:discover\"",
	"ST:ssdp:all",
	"MX:1",
}

type Bridge struct {
	ip net.IP
}

// Discover Hue bridges via SSDP.
// Returns a map of IP.String() to empty struct.
func Discover() ([]string, error) {
	rAddr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		return nil, err
	}

	conn, err := net.DialUDP("udp4", nil, rAddr)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	lAddr, err := net.ListenMulticastUDP("udp4", nil, rAddr)
	if err != nil {
		return nil, err
	}
	defer lAddr.Close()

	// Write discovery packet to network
	if _, err = conn.Write([]byte(strings.Join(_SSDPData, "\r\n"))); err != nil {
		return nil, err
	}

	// Read responses back for short time period
	timeoutDuration := _DefaultTimeout
	var buf bytes.Buffer
	buf.Grow(_DefaultBufferSize)

	bridgeSet := make([]string, 0, _DefaultNumBridges)

	for {
		lAddr.SetReadDeadline(time.Now().Add(timeoutDuration))
		buf.Reset()
		n, addr, err := lAddr.ReadFromUDP(buf.Bytes())
		if err != nil {
			switch osErr := err.(*net.OpError); {
			case osErr.Timeout():
				// Timeout
			case osErr.Temporary():
				// Transient condition
			default:
				// Return what we have
				return bridgeSet, err
			}
		}

		if bytes.Contains(buf.Bytes()[:n], []byte(_SSDPIdentifier)) {
			bridgeSet = append(bridgeSet, addr.IP.String())
		}
	}

	return bridgeSet, nil
}
