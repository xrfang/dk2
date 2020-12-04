package ctrl

import (
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)

type (
	token struct {
		ex time.Time
		ip net.IP
	}
	tokens struct {
		reg map[string]*token
		exp time.Duration
		sync.Mutex
	}
)

var TS tokens

func init() {
	rand.Seed(time.Now().UnixNano())
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

func initTokenStore(ttl int) {
	TS.Lock()
	defer TS.Unlock()
	TS.reg = make(map[string]*token)
	TS.exp = time.Duration(ttl) * time.Second
	go func() {
		for {
			time.Sleep(time.Minute)
			now := time.Now()
			func() {
				TS.Lock()
				defer TS.Unlock()
				for t, k := range TS.reg {
					if now.After(k.ex) {
						delete(TS.reg, t)
					}
				}
			}()
		}
	}()
}

func (ts *tokens) Set(r *http.Request) (string, time.Time) {
	tok := uuid(8)
	exp := time.Now().Add(ts.exp)
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ts.Lock()
	defer ts.Unlock()
	ts.reg[tok] = &token{ex: exp, ip: net.ParseIP(host)}
	return tok, exp
}

func (ts *tokens) Get(r *http.Request) bool {
	tok := r.URL.Query().Get("t")
	if tok == "" {
		tok = getCookie(r, "t")
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	ip := net.ParseIP(host)
	ts.Lock()
	defer ts.Unlock()
	t := ts.reg[tok]
	return t != nil && t.ip != nil && t.ip.Equal(ip) && time.Now().Before(t.ex)
}
