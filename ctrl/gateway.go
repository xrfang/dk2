package ctrl

import (
	"dk/base"
	"fmt"
	"net"
	"time"
)

func Start(cf Config) {
	initAdapterManager(cf)
	startAdminInterface(cf)
	startBackendRegistrar(cf)
	handshake := time.Duration(cf.Handshake) * time.Second
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", cf.ServPort))
	assert(err)
	for {
		conn, err := ln.Accept()
		if err != nil {
			base.Log("accept: %v", err)
			time.Sleep(time.Second)
			continue
		}
		go func(c net.Conn) {
			ra := c.RemoteAddr().String()
			assert(c.SetDeadline(time.Now().Add(handshake)))
			buf := make([]byte, 32)
			n, err := c.Read(buf)
			if err != nil {
				err, ok := err.(net.Error)
				if !ok || !err.Timeout() {
					base.Dbg("validate(%s): %v", ra, err)
				} else {
					base.Dbg("validate(%s): %v", ra, err)
				}
				base.Log(`backend "%s" refused (handshake failed)`, ra)
				c.Close()
				return
			}
			assert(c.SetReadDeadline(time.Time{}))
			name := func(mac []byte) string {
				if len(mac) != 32 {
					return ""
				}
				for name, key := range cf.Auths {
					var match bool
					res := base.Authenticate(mac[:16], name, key)
					for i, c := range mac {
						match = res[i] == c
						if !match {
							break
						}
					}
					if match {
						return name
					}
				}
				return ""
			}(buf[:n])
			if name == "" {
				base.Dbg("validate(%s): invalid hmac [%x]", ra, buf[:n])
				base.Log(`backend "%s" refused (handshake failed)`, ra)
				c.Close()
				return
			}
			base.Log(`backend "%s" connected (%s)`, ra, name)
			br <- reqServ{name, c}
		}(conn)
	}
}
