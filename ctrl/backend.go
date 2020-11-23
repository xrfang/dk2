package ctrl

import "net"

type backend struct {
	cin  net.Conn
	cout map[string]net.Conn
}

type backends map[string]backend
