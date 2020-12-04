package ctrl

import (
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/pquerna/otp/totp"
)

var (
	allowed func(r *http.Request) bool
	pid     string
)

func init() {
	pid = strconv.Itoa(os.Getpid())
}

func authChk(r *http.Request, cf Config) (string, bool) {
	usr := r.URL.Query().Get("u")
	if usr == "" {
		usr = getCookie(r, "u")
	}
	otp := r.URL.Query().Get("p")
	key := cf.Users[usr]
	if totp.Validate(otp, key) {
		return usr, true
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", false
	}
	return "", pid == otp
}

func setWatchdog(cf Config) {
	initTokenStore(cf.AuthTime)
	allowed = func(r *http.Request) bool {
		if TS.Get(r) {
			return true
		}
		_, ok := authChk(r, cf)
		return ok
	}
}
