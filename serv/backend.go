package serv

import (
	"dk/base"
	"fmt"
	"net"
	"time"
)

func Start(cf Config) {
	addr := fmt.Sprintf("%s:%d", cf.CtrlHost, cf.CtrlPort)
	for {
		func() {
			d := net.Dialer{Timeout: time.Duration(cf.ConnWait) * time.Second}
			conn, err := d.Dial("tcp", addr)
			if err != nil {
				base.Log("%v", err)
				return
			}
			base.Log("connected to %s", addr)
			handshake := base.Authenticate(nil, cf.Name, cf.Auth)
			_, err = conn.Write(handshake)
			if err != nil {
				base.Log("%v", err)
				return
			}
			serve(conn)
		}()
		time.Sleep(time.Second)
	}
}
