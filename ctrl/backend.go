package ctrl

import (
	"dk/base"
	"encoding/binary"
	"net"
	"time"
)

const (
	queueCap = 1024 //包处理队列长度
)

type (
	chunk struct {
		cls base.ChunkType
		buf []byte
		arg interface{}
	}
	backend struct {
		serv net.Conn
		comm chan chunk
		clis map[uint32]*base.Conn
	}
	backends map[string]*backend
	reqServ  struct {
		name string
		conn net.Conn
	}
	reqConn struct {
		session uint32
		backend string
		host    net.IP
		port    uint16
		conn    net.Conn
	}
)

func (b *backend) Remove(session uint32) {
	s := b.clis[session]
	if s == nil {
		return
	}
	s.Close()
	base.Dbg("removed session %x", session)
	delete(b.clis, session)
}

func (b *backend) Free() {
	if b.serv != nil {
		b.serv.Close()
	}
	for _, c := range b.clis {
		c.Close()
	}
}

func NewBackend(name string, conn net.Conn, cf Config) *backend {
	b := &backend{
		serv: conn,
		comm: make(chan chunk, queueCap),
		clis: make(map[uint32]*base.Conn),
	}
	if cf.KeepAlive > 0 { //定时PING后端，保持连接不被NAT防火墙关闭
		go func() {
			ping := time.Duration(cf.KeepAlive) * time.Second
			for {
				time.Sleep(ping)
				if err := base.Ping(conn); err != nil {
					base.Log("ping(%s): %v", name, err)
					return
				}
			}
		}()
	}
	go func() {
		for {
			var session uint32
			var data []byte
			c := <-b.comm
			if len(c.buf) >= 4 {
				session = binary.BigEndian.Uint32(c.buf[:4])
				data = c.buf[4:]
			}
			switch c.cls {
			case base.ChunkCLS:
				b.Remove(session)
			case base.ChunkDAT:
				s := b.clis[session]
				if s == nil {
					base.Dbg("dispatch[%x]: session not found, dropped %d bytes", session, len(data))
					break
				}
				if err := s.Send(base.ChunkDAT, data); err != nil {
					base.Log("dispatch[%x]: %v", session, err)
					if err != base.ErrInvalidChunk {
						b.Remove(session)
					}
				}
			case base.ChunkCMD:
				switch data[0] {
				case 0:
					base.Dbg("received pong from backend")
				}
			case base.ChunkNUL:
				if session == 0 { //清理空闲连接
					base.Dbg("[%s] cleaning %d remote sessions", name, len(b.clis))
					for s, c := range b.clis {
						if c.Idle(cf.IdleClose) {
							base.Dbg("[%s] closing idle session %x", name, s)
							c.Close()
							delete(b.clis, s)
						}
					}
					break
				}
				//创建新连接
				conn, ok := c.arg.(net.Conn)
				if !ok {
					base.Log("[%s] invalid arg type: %T", name, c.arg)
					break
				}
				b.Remove(session)
				b.clis[session] = base.NewConn(conn)
				base.Open(b.serv, session, data)
				go func(c net.Conn) {
					defer func() {
						if e := recover(); e != nil {
							base.Dbg("[%s] session %x: %v", name, session, e)
							base.Close(b.serv, session)
						}
					}()
					data := make([]byte, base.MTU-2)
					for {
						n, err := c.Read(data)
						assert(err)
						assert(base.Send(b.serv, data[:n]))
					}
				}(conn)
			}
		}
	}()
	go func() { //从后端接收数据，分发给客户端
		for {
			ct, buf, err := base.Recv(conn)
			if err != nil {
				base.Log("recv: %v", err)
				base.Dbg(`unregister backend "%s"`, name)
				ch <- reqServ{name, nil}
				return
			}
			b.comm <- chunk{ct, buf, nil}
		}
	}()
	go func() { //定时清理不活跃的前端连接，释放系统资源
		interval := cf.IdleClose / 2
		if interval < 60 {
			interval = 60
		}
		if interval > 600 {
			interval = 600
		}
		for {
			time.Sleep(time.Duration(interval) * time.Second)
			b.comm <- chunk{base.ChunkNUL, nil, nil}
		}
	}()
	return b
}

var (
	ch chan interface{}
	bs backends
)

func init() {
	bs = make(backends)
	ch = make(chan interface{}, 64)
}

func startBackendRegistrar(cf Config) {
	go func() {
		for {
			cmd := <-ch
			switch cmd.(type) {
			case reqServ:
				req := cmd.(reqServ)
				//不论是注册后端还是删除后端，都需要先清理一下注册记录
				b := bs[req.name]
				if b != nil {
					b.Free()
				}
				delete(bs, req.name)
				if req.conn != nil { //conn非空，表示注册新后端
					bs[req.name] = NewBackend(req.name, req.conn, cf)
				}
			case reqConn:
				//req := cmd.(reqConn)

			}
		}
	}()
}
