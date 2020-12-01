package ctrl

import (
	"net"
	"net/http"
	"time"
)

func apiAuth(w http.ResponseWriter, r *http.Request) {
	if !allowed(r) {
		return
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	if ip == nil {
		jsonReply(w, map[string]interface{}{
			"stat": false,
			"mesg": "bad request",
		})
		return
	}
	ch := make(chan interface{})
	das.ch <- authReq{from: ip, port: 0, rply: ch}
	select {
	case rep := <-ch:
		jsonReply(w, map[string]interface{}{
			"stat": true,
			"data": rep,
		})
	case <-time.After(chanLife):
		jsonReply(w, map[string]interface{}{
			"stat": false,
			"mesg": "no reply",
		})
	}
}
