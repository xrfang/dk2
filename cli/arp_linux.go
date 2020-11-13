//+build linux

package cli

import (
	"fmt"
	"net"
	"syscall"
	"time"
)

func findInterface(dstIP net.IP) (net.IP, *net.Interface, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, nil, err
	}
	for _, i := range ifs {
		if i.Flags&1 == 0 { //interface down
			continue
		}
		addrs, err := i.Addrs()
		if err != nil {
			continue //cannot get interface addresses
		}
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ipnet.Contains(dstIP) {
					return ipnet.IP, &i, nil
				}
			}
		}
	}
	return nil, nil, nil
}

func getMAC(ip string, timeout time.Duration) (mac string, err error) {
	addr := net.ParseIP(ip)
	if len(addr.To4()) != net.IPv4len {
		return "", fmt.Errorf("not valid IPv4 address: %v", ip)
	}
	srcIP, ifce, err := findInterface(addr)
	if ifce == nil {
		return "", err
	}

	toSockAddr := syscall.SockaddrLinklayer{Ifindex: ifce.Index}
	const proto = 1544 //htons(ETH_P_ARP)
	sock, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, proto)
	if err != nil {
		return "", err
	}
	defer syscall.Close(sock)

	srcMac := ifce.HardwareAddr
	broadcastMac := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	req := newArpRequest(srcMac, srcIP, broadcastMac, addr)

	type PingResult struct {
		mac net.HardwareAddr
		err error
	}
	rc := make(chan PingResult, 1)
	go func() {
		err := syscall.Sendto(sock, req.MarshalWithEthernetHeader(), 0, &toSockAddr)
		if err != nil {
			rc <- PingResult{nil, err}
			return
		}
		for {
			buf := make([]byte, 128)
			n, _, err := syscall.Recvfrom(sock, buf, 0)
			if err != nil {
				rc <- PingResult{nil, err}
				return
			}
			rep := parseArpDatagram(buf[14:n])
			if rep.IsResponseOf(req) {
				rc <- PingResult{rep.SenderMac(), nil}
				return
			}
		}
	}()
	var pr PingResult
	select {
	case pr = <-rc:
	case <-time.After(timeout):
		pr = PingResult{nil, fmt.Errorf("arping timeout: %s", ip)}
	}
	if pr.err != nil {
		return "", pr.err
	}
	return pr.mac.String(), nil
}
