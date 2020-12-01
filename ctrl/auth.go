package ctrl

import (
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/pquerna/otp/totp"
)

var (
	allowed func(r *http.Request) bool
	tokens  sync.Map
	pid     string
)

func init() {
	pid = strconv.Itoa(os.Getpid())
}

func uuid(L int) string {
	cs := "0123456789abcdefghijklmnopqrstuvwxyz"
	buf := make([]byte, L)
	rand.Read(buf)
	for i := 0; i < L; i++ {
		buf[i] = cs[buf[i]%36]
	}
	return string(buf)
}

func authChk(r *http.Request, cf Config) bool {
	otp := r.URL.Query().Get("p")
	key := cf.Users[r.URL.Query().Get("u")]
	if totp.Validate(otp, key) {
		return true
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return false
	}
	return pid == otp
}

func setWatchdog(cf Config) {
	allowed = func(r *http.Request) bool {
		token := r.URL.Query().Get("t")
		if token == "" {
			token = getCookie(r, "t")
		}
		if token != "" {
			exp, ok := tokens.Load(token)
			if !ok || time.Now().After(exp.(time.Time)) {
				return false
			}
			return true
		}
		return authChk(r, cf)
	}
	go func() {
		for {
			time.Sleep(time.Minute)
			now := time.Now()
			tokens.Range(func(k, v interface{}) bool {
				if now.After(v.(time.Time)) {
					tokens.Delete(k)
				}
				return true
			})
		}
	}()
}
