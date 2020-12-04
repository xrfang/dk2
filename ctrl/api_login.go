package ctrl

import (
	"net/http"
	"time"
)

func apiLogin(cf Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if usr, ok := authChk(r, cf); ok {
			tok, exp := TS.Set(r)
			setCookie(w, "t", tok, cf.AuthTime)
			if usr != "" {
				setCookie(w, "u", usr, 365*86400)
			}
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
