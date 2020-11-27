package ctrl

import (
	"net/http"
	"time"
)

func apiSite(w http.ResponseWriter, r *http.Request) {
	ch := make(chan interface{})
	br <- reqList{ch}
	select {
	case rep := <-ch:
		jsonReply(w, rep)
	case <-time.After(chanLife):
		http.Error(w, "Timeout", http.StatusGatewayTimeout)
	}
}
