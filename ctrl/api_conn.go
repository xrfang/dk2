package ctrl

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
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
	host := "127.0.0.1"
	if len(p) == 3 && len(p[2]) > 0 {
		ip := net.ParseIP(p[2])
		if ip == nil {
			jsonReply(w, map[string]interface{}{
				"stat": false,
				"mesg": fmt.Sprintf("host '%s' is not valid IP", p[2]),
			})
			return
		}
		host = ip.String()
	}
	fmt.Printf("conn: name=%s; port=%d; host=%s\n", name, port, host)
}
