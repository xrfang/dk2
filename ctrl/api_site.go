package ctrl

import (
	"net/http"
	"time"
)

func apiSite(w http.ResponseWriter, r *http.Request) {
	if !allowed(r) {
		return
	}
	ch := make(chan interface{})
	br <- reqList{"", ch}
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
