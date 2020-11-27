package ctrl

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func apiScan(w http.ResponseWriter, r *http.Request) {
	p := strings.SplitN(r.URL.Path[9:], "/", 2)
	if len(p) != 2 {
		jsonReply(w, map[string]interface{}{
			"stat": false,
			"mesg": "name/port expected",
		})
		return
	}
	port, _ := strconv.Atoi(p[1])
	if port <= 0 || port > 65535 {
		jsonReply(w, map[string]interface{}{
			"stat": false,
			"mesg": fmt.Sprintf("invalid port '%s', 1~65535 expected", p[1]),
		})
		return
	}
	ch := make(chan interface{})
	br <- reqScan{name: p[0], port: uint16(port), rep: ch}
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
