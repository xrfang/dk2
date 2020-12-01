package ctrl

import (
	"net/http"
	"time"
)

func apiLogin(cf Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if authChk(r, cf) {
			tok := uuid(8)
			exp := time.Now().Add(time.Duration(cf.AuthTime) * time.Second)
			setCookie(w, "t", tok, cf.AuthTime)
			tokens.Store(tok, exp)
			jsonReply(w, map[string]interface{}{
				"stat": true,
				"data": map[string]interface{}{
					"token": tok,
					"valid": exp.Format(time.RFC3339),
				},
			})
		} else {
			jsonReply(w, map[string]interface{}{
				"stat": false,
				"mesg": "access denied",
			})
		}
	}
}
