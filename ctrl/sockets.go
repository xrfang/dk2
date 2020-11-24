package ctrl

import (
	"dk/base"
	"encoding/binary"
	"net"
	"sync"
	"time"
)

type (
	socketRegistry struct {
		pool map[string]*base.Conn //维护所有后端连接，索引为后端名称
		peer map[uint32]*base.Conn //维护所有客户连接，索引为SESSION-ID
		sync.Mutex
	}
)

var sr socketRegistry

func init() {
	sr = socketRegistry{
		pool: make(map[string]*base.Conn),
		peer: make(map[uint32]*base.Conn),
	}
}

func unregisterBackend(name string) {
	sr.Lock()
	defer sr.Unlock()
	base.Dbg(`unregister backend "%s"`, name)
	c := sr.pool[name]
	if c != nil {
		c.Close()
	}
	delete(sr.pool, name)
}

func dispatch(session uint32, data []byte) {
	sr.Lock()
	defer sr.Unlock()
	c := sr.peer[session]
	if c == nil {
		base.Dbg("dispatch[%x]: session not found, dropped %d bytes", session, len(data))
		return
	}
	if err := c.Send(0, base.ChunkDAT, data); err != nil {
		base.Log("dispatch[%x]: %v", session, err)
	}
}

func finish(session uint32) {
	sr.Lock()
	defer sr.Unlock()
	c := sr.peer[session]
	if c == nil {
		base.Dbg("session %x not found, cannot finish", session)
		return
	}
	c.Close()
	delete(sr.peer, session)
	base.Dbg("backend finished session %x", session)
}

func RegisterBackend(name string, conn net.Conn, cf Config) {
	sr.Lock()
	defer sr.Unlock()
	dc := base.NewConn(conn)
	sr.pool[name] = dc
	//定时PING后端，保持连接不被NAT防火墙关闭
	go func() {
		if cf.KeepAlive <= 0 {
			return
		}
		ping := time.Duration(cf.KeepAlive) * time.Second
		defer func() {
			if e := recover(); e != nil {
				base.Log("ping: %v", e)
				unregisterBackend(name)
			}
		}()
		for {
			time.Sleep(ping)
			assert(dc.Send(0, base.ChunkCMD, []byte{0}))
		}
	}()
	//定时清理不活跃的前端连接，释放系统资源
	go func() {
		interval := cf.IdleClose / 2
		if interval < 60 {
			interval = 60
		}
		if interval > 600 {
			interval = 600
		}
		for {
			time.Sleep(time.Duration(interval) * time.Second)
			func() {
				sr.Lock()
				defer sr.Unlock()
				for s, p := range sr.peer {
					if p.Idle(cf.IdleClose) {
						p.Close()
						delete(sr.peer, s)
					}
				}
			}()
		}
	}()
	//从后端接收数据，分发给客户端
	go func() {
		defer func() {
			if e := recover(); e != nil {
				base.Log("recv: %v", e)
				unregisterBackend(name)
			}
		}()
		for {
			ct, buf, err := dc.Recv(0)
			assert(err)
			switch ct {
			case base.ChunkCLS:
				session := binary.BigEndian.Uint32(buf[:4])
				go finish(session)
			case base.ChunkDAT:
				session := binary.BigEndian.Uint32(buf[:4])
				go dispatch(session, buf[4:])
			case base.ChunkCMD:
				switch buf[0] {
				case 0:
					base.Dbg("received pong from backend")
				}
			}
		}
	}()
}
