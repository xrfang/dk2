package ctrl

import (
	"dk/base"
	"encoding/binary"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"
)

type (
	dkAdapter struct {
		wire net.Listener
		port uint16
		auth map[string]*authReq //来源IP=>目标的映射
		used time.Time           //最后使用时间
		sync.RWMutex
	}
	authReq struct {
		from net.IP
		name string
		host net.IP
		port uint16
		time time.Time //过期时间
		rply chan interface{}
	}
	dkAdapters struct {
		up time.Time //上次清理AUTH的时间
		as map[uint16]*dkAdapter
		ch chan authReq
	}
)

func (da *dkAdapter) getAuth(from net.IP) *authReq {
	da.RLock()
	defer da.RUnlock()
	return da.auth[from.String()]
}

func (da *dkAdapter) setAuth(a *authReq) {
	da.Lock()
	defer da.Unlock()
	da.auth[a.from.String()] = a
}

func (da *dkAdapter) refreshAuths() {
	da.Lock()
	defer da.Unlock()
	for s, a := range da.auth {
		if time.Now().After(a.time) {
			base.Dbg("adapter#%d: removed expired auth for %s", da.port, s)
			delete(da.auth, s)
		}
	}
}

func (da *dkAdapter) Used() {
	da.Lock()
	da.used = time.Now()
	da.Unlock()
}

func (da *dkAdapter) IsIdle() bool {
	da.RLock()
	defer da.RUnlock()
	return time.Since(da.used) >= adapterIdleLife
}

func (da *dkAdapter) Match(ar authReq) int {
	d := da.getAuth(ar.from)
	if d == nil {
		return 0 //该接口没有与来源src匹配的授权
	}
	if d.name != ar.name || !d.host.Equal(ar.host) || d.port != ar.port {
		return -1 //该接口与来源src匹配的授权与dst不符
	}
	return 1 //找到授权匹配
}

func (da *dkAdapter) RequestConnection(conn net.Conn) {
	addr := conn.RemoteAddr().(*net.TCPAddr)
	from := addr.IP
	ar := da.getAuth(from)
	if ar == nil || time.Now().After(ar.time) {
		base.Log("[adapter#%d] cannot get auth for %s", da.port, from.String())
		conn.Close()
		return
	}
	dest := make([]byte, 2)
	binary.BigEndian.PutUint16(dest, ar.port)
	dest = append(dest, ar.host...)
	br <- reqConn{
		session: rand.Uint32(),
		backend: ar.name,
		dest:    dest,
		conn:    conn,
	}
	da.Used()
}

func newAdapter(serv uint16, ar *authReq) (da *dkAdapter, err error) {
	defer func() {
		if e := recover(); e != nil {
			err = e.(error)
		}
	}()
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", serv))
	assert(err)
	da = &dkAdapter{
		wire: ln,
		port: serv,
		auth: map[string]*authReq{ar.from.String(): ar},
		used: time.Now(),
	}
	go func() {
		defer func() {
			if e := recover(); e != nil {
				base.Log("accept(%d): %v", serv, e)
			}
			ln.Close()                               //只要退出本函数，一定需要关闭listener
			das.ch <- authReq{from: nil, port: serv} //通知管理器删除该接口
		}()
		tl := ln.(*net.TCPListener)
		for {
			assert(tl.SetDeadline(time.Now().Add(time.Second)))
			conn, err := tl.Accept()
			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					if da.IsIdle() {
						break
					}
					continue
				}
				panic(err)
			}
			go da.RequestConnection(conn)
		}
	}()
	return
}

//RefreshAuths 清理过期的授权
func (das *dkAdapters) RefreshAuths() {
	if time.Since(das.up) < time.Minute {
		return //最多每分钟清理一次
	}
	for _, da := range das.as {
		da.refreshAuths()
	}
}

var (
	das             dkAdapters
	adapterIdleLife time.Duration
)

func initAdapterManager(cf Config) {
	adapterIdleLife = time.Duration(cf.AuthTime) * time.Second
	das.as = make(map[uint16]*dkAdapter)
	das.ch = make(chan authReq, 16)
	go func() {
		bq := make(chan interface{})
		for {
		serv:
			ar := <-das.ch
			das.RefreshAuths()
			if ar.from == nil { //表示为adapter空闲超时关闭，需要剔除
				delete(das.as, ar.port)
				continue
			}
			if ar.port == 0 { //表示查询给来源为"from"的所有授权
				auths := []map[string]interface{}{}
				for p, da := range das.as {
					a := da.getAuth(ar.from)
					if a != nil {
						auths = append(auths, map[string]interface{}{
							"port":  p,
							"site":  a.name,
							"addr":  fmt.Sprintf("%s:%d", a.host, a.port),
							"until": a.time.Format(time.RFC3339),
						})
					}
				}
				ar.rply <- auths
				continue
			}
			ar.time = time.Now().Add(time.Duration(cf.AuthTime) * time.Second)
			br <- reqList{name: ar.name, rep: bq}
			select {
			case rep := <-bq:
				r := rep.(map[string]interface{})
				bl := r["data"].([]map[string]interface{})
				if len(bl) == 0 {
					ar.rply <- -2
					goto serv
				}
			case <-time.After(chanLife):
				base.Log("queryBackend(%s): timeout", ar.name)
				ar.rply <- -1
				goto serv
			}
			var fa *dkAdapter
			for p, da := range das.as {
				switch da.Match(ar) {
				case 0:
					if fa == nil {
						fa = da
					}
				case 1:
					ar.rply <- int(p)
					goto serv
				}
			}
			if fa != nil { //没有找到匹配的接口，但有空闲的接口可用
				fa.setAuth(&ar)
				ar.rply <- int(fa.port)
				continue
			}
			//没有空闲接口，创建一个新接口
			if len(das.as) >= cf.MaxServes {
				ar.rply <- 0 //接口数量已经到达上限
				continue
			}
			for p := uint16(cf.ServPort + 1); p <= 65535; p++ {
				if das.as[p] == nil {
					na, err := newAdapter(p, &ar)
					if err == nil {
						das.as[p] = na
						ar.rply <- int(p)
					} else {
						base.Log("newAdapter(%d, %s): %v", p, ar.from, err)
						ar.rply <- 0 //创建新接口失败
					}
					break
				}
			}
		}
	}()
}
