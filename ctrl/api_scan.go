package ctrl

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

func apiScan(w http.ResponseWriter, r *http.Request) {
	p := strings.SplitN(r.URL.Path[9:], "/", 2)
	if len(p) != 2 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	port, _ := strconv.Atoi(p[1])
	if port <= 0 || port > 65535 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	ch := make(chan interface{})
	br <- reqScan{name: p[0], port: uint16(port), rep: ch}
	select {
	case rep := <-ch:
		jsonReply(w, rep)
	case <-time.After(chanLife):
		http.Error(w, "Timeout", http.StatusGatewayTimeout)
	}
}
