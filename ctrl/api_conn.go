package ctrl

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func apiConn(w http.ResponseWriter, r *http.Request) {
	p := strings.Split(r.URL.Path[9:], "/")
	if len(p) < 2 || len(p) > 3 {
		jsonReply(w, map[string]interface{}{
			"stat": false,
			"mesg": "name/port[/ip] expected",
		})
		return
	}
	name := p[0]
	port, _ := strconv.Atoi(p[1])
	if port <= 0 || port > 65535 {
		jsonReply(w, map[string]interface{}{
			"stat": false,
			"mesg": fmt.Sprintf("invalid port '%s', 1~65535 expected", p[1]),
		})
		return
	}
	host := net.ParseIP("127.0.0.1")
	if len(p) == 3 && len(p[2]) > 0 {
		ip := net.ParseIP(p[2])
		if ip == nil {
			jsonReply(w, map[string]interface{}{
				"stat": false,
				"mesg": fmt.Sprintf("host '%s' is not valid IP", p[2]),
			})
			return
		}
		host = ip
	}
	rip, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(rip)
	if ip == nil {
		jsonReply(w, map[string]interface{}{
			"stat": false,
			"mesg": fmt.Sprintf("failed to parse remote addr '%s'", r.RemoteAddr),
		})
		return
	}
	ch := make(chan int)
	das.ch <- authReq{
		from: ip,
		name: name,
		host: host,
		port: uint16(port),
		rply: ch,
	}
	select {
	case rep := <-ch:
		jsonReply(w, rep)
	case <-time.After(chanLife):
		jsonReply(w, map[string]interface{}{
			"stat": false,
			"mesg": "no reply",
		})
	}
}
