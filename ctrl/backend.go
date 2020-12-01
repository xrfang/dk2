package ctrl

import (
	"dk/base"
	"encoding/binary"
	"encoding/json"
	"fmt"
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
	reqServ  struct { //后端注册
		name string
		conn net.Conn
	}
	reqConn struct { //前端连接
		session uint32
		backend string
		dest    []byte //格式：大端序uint16端口号+net.IP格式的目标IP
		conn    net.Conn
	}
	reqList struct { //列出指定后端，若name为空，列出所有后端
		name string
		rep  chan interface{}
	}
	reqScan struct { //扫描后端开放某端口的主机
		name string
		port uint16
		rep  chan interface{}
	}
	repScan struct { //端口扫描的回复
		sid uint32
		msg map[string]interface{}
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
				if err := s.Send(data); err != nil {
					base.Log("dispatch[%x]: %v", session, err)
					b.Remove(session)
				}
			case base.ChunkCMD:
				switch data[0] {
				case 0:
					base.Dbg("received pong from backend")
				case 1:
					var rep map[string]interface{}
					json.Unmarshal(data[1:], &rep)
					br <- repScan{session, rep}
				}
			case base.ChunkCON:
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
							buf := make([]byte, 4)
							binary.BigEndian.PutUint32(buf, session)
							b.comm <- chunk{base.ChunkCLS, buf, nil}
						}
					}()
					data := make([]byte, base.MTU-2)
					for {
						n, err := c.Read(data)
						assert(err)
						assert(base.Send(b.serv, session, data[:n]))
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
				br <- reqServ{name, nil}
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
			b.comm <- chunk{base.ChunkCON, nil, nil}
		}
	}()
	return b
}

var (
	br chan interface{}
	bs backends
)

func init() {
	bs = make(backends)
	br = make(chan interface{}, 64)
}

func startBackendRegistrar(cf Config) {
	go func() {
		for {
			cmd := <-br
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
				req := cmd.(reqConn)
				b := bs[req.backend]
				if b == nil {
					req.conn.Close()
					break
				}
				buf := make([]byte, 4)
				binary.BigEndian.PutUint32(buf, req.session)
				buf = append(buf, req.dest...)
				b.comm <- chunk{base.ChunkCON, buf, req.conn}
			case reqList:
				req := cmd.(reqList)
				list := []map[string]interface{}{}
				for n, b := range bs {
					if req.name != "" && req.name != n {
						continue
					}
					conns := []string{}
					for _, c := range b.clis {
						addr := c.Remote()
						if addr != nil {
							conns = append(conns, addr.IP.String())
						}
					}
					list = append(list, map[string]interface{}{
						"name": n,
						"conn": conns,
					})
				}
				req.rep <- map[string]interface{}{
					"stat": true,
					"data": list,
				}
			case reqScan:
				req := cmd.(reqScan)
				b := bs[req.name]
				if b == nil {
					req.rep <- map[string]interface{}{
						"stat": false,
						"mesg": fmt.Sprintf("backend '%s' not found", req.name),
					}
					break
				}
				buf := make([]byte, 3)
				buf[0] = 1
				binary.BigEndian.PutUint16(buf[1:], req.port)
				cid := setChan(req.rep)
				base.Reply(b.serv, cid, buf)
			case repScan:
				rep := cmd.(repScan)
				ch := getChan(rep.sid)
				if ch != nil {
					ch <- rep.msg
				}
			}
		}
	}()
}
